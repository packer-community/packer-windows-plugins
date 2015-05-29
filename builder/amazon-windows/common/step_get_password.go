package common

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/awslabs/aws-sdk-go/service/ec2"
	"github.com/mitchellh/multistep"
	"github.com/mitchellh/packer/packer"

	wincommon "github.com/packer-community/packer-windows-plugins/common"
)

type StepGetPassword struct {
	WinRMConfig        *wincommon.WinRMConfig
	RunConfig          *RunConfig
	GetPasswordTimeout time.Duration
}

func (s *StepGetPassword) Run(state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packer.Ui)
	instance := state.Get("instance").(*ec2.Instance)

	if s.RunConfig.NewAdministratorPassword != "" {
		s.WinRMConfig.WinRMPassword = s.RunConfig.NewAdministratorPassword
		return multistep.ActionContinue
	}

	var password string
	var err error

	cancel := make(chan struct{})
	waitDone := make(chan bool, 1)
	go func() {
		ui.Say(fmt.Sprintf("Retrieving auto-generated password for instance %s...", *instance.InstanceID))

		password, err = s.waitForPassword(state, cancel)
		if err != nil {
			waitDone <- false
			return
		}
		waitDone <- true
	}()

	log.Printf("Waiting to retrieve instance %s password, up to timeout: %s", *instance.InstanceID, s.GetPasswordTimeout)
	timeout := time.After(s.GetPasswordTimeout)

WaitLoop:
	for {
		// Wait for one of: the password becoming available, a timeout occuring
		// or an interrupt coming through.
		select {
		case <-waitDone:
			if err != nil {
				state.Put("error", err)
				ui.Error(err.Error())
				return multistep.ActionHalt
			}

			s.WinRMConfig.WinRMPassword = password
			break WaitLoop

		case <-timeout:
			err := fmt.Errorf(fmt.Sprintf("Timeout retrieving password for instance %s", *instance.InstanceID))
			state.Put("error", err)
			ui.Error(err.Error())
			close(cancel)
			return multistep.ActionHalt

		case <-time.After(1 * time.Second):
			if _, ok := state.GetOk(multistep.StateCancelled); ok {
				// Build was cancelled.
				close(cancel)
				log.Println("Interrupt detected, cancelling password retrieval")
				return multistep.ActionHalt
			}
		}
	}

	return multistep.ActionContinue

}

func (s *StepGetPassword) waitForPassword(state multistep.StateBag, cancel <-chan struct{}) (string, error) {
	ec2conn := state.Get("ec2").(*ec2.EC2)
	instance := state.Get("instance").(*ec2.Instance)
	privateKey := state.Get("privateKey").(string)

	for {
		select {
		case <-cancel:
			log.Println("Retrieve password wait cancelled. Exiting loop.")
			return "", errors.New("Retrieve password wait cancelled")

		case <-time.After(20 * time.Second):
		}

		input := &ec2.GetPasswordDataInput{
			InstanceID: instance.InstanceID,
		}
		resp, err := ec2conn.GetPasswordData(input)
		if err != nil {
			err := fmt.Errorf("Error retrieving auto-generated instance password: %s", err)
			return "", err
		}

		if *resp.PasswordData != "" {
			decryptedPassword, err := decryptPasswordDataWithPrivateKey(*resp.PasswordData, []byte(privateKey))
			if err != nil {
				err := fmt.Errorf("Error decrypting auto-generated instance password: %s", err)
				return "", err
			}
			return decryptedPassword, nil
		}
	}
}

func (s *StepGetPassword) Cleanup(multistep.StateBag) {
	// No cleanup...
}

func decryptPasswordDataWithPrivateKey(passwordData string, pemBytes []byte) (string, error) {
	encryptedPasswd, err := base64.StdEncoding.DecodeString(passwordData)
	if err != nil {
		return "", err
	}

	block, _ := pem.Decode(pemBytes)
	var asn1Bytes []byte
	if _, ok := block.Headers["DEK-Info"]; ok {
		return "", fmt.Errorf("Cannot decrypt instance password as the keypair is protected with a passphrase")
	}

	asn1Bytes = block.Bytes

	key, err := x509.ParsePKCS1PrivateKey(asn1Bytes)
	if err != nil {
		return "", err
	}

	out, err := rsa.DecryptPKCS1v15(nil, key, encryptedPasswd)
	if err != nil {
		return "", err
	}

	return string(out), nil
}
