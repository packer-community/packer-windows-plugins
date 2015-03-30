package common

import (
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"time"

	"github.com/mitchellh/goamz/ec2"
	"github.com/mitchellh/multistep"
	"github.com/mitchellh/packer/packer"

	awscommon "github.com/mitchellh/packer/builder/amazon/common"
)

type StepRunSourceInstance struct {
	RunConfig          *RunConfig
	BlockDevices       *awscommon.BlockDevices
	ExpectedRootDevice string
	Debug              bool

	instance    *ec2.Instance
	spotRequest *ec2.SpotRequestResult
}

func (s *StepRunSourceInstance) Run(state multistep.StateBag) multistep.StepAction {
	ec2conn := state.Get("ec2").(*ec2.EC2)
	keyName := state.Get("keyPair").(string)
	securityGroupIds := state.Get("securityGroupIds").([]string)
	ui := state.Get("ui").(packer.Ui)

	userData := s.RunConfig.UserData
	if s.RunConfig.UserDataFile != "" {
		contents, err := ioutil.ReadFile(s.RunConfig.UserDataFile)
		if err != nil {
			state.Put("error", fmt.Errorf("Problem reading user data file: %s", err))
			return multistep.ActionHalt
		}

		userData = string(contents)
	}

	securityGroups := make([]ec2.SecurityGroup, len(securityGroupIds))
	for n, securityGroupId := range securityGroupIds {
		securityGroups[n] = ec2.SecurityGroup{Id: securityGroupId}
	}

	ui.Say("Launching a source AWS instance...")
	imageResp, err := ec2conn.Images([]string{s.SourceAMI}, ec2.NewFilter())
	if err != nil {
		state.Put("error", fmt.Errorf("There was a problem with the source AMI: %s", err))
		return multistep.ActionHalt
	}

	if len(imageResp.Images) != 1 {
		state.Put("error", fmt.Errorf("The source AMI '%s' could not be found.", s.SourceAMI))
		return multistep.ActionHalt
	}

	if s.ExpectedRootDevice != "" && imageResp.Images[0].RootDeviceType != s.ExpectedRootDevice {
		state.Put("error", fmt.Errorf(
			"The provided source AMI has an invalid root device type.\n"+
				"Expected '%s', got '%s'.",
			s.ExpectedRootDevice, imageResp.Images[0].RootDeviceType))
		return multistep.ActionHalt
	}

	spotPrice := s.SpotPrice
	if spotPrice == "auto" {
		ui.Message(fmt.Sprintf(
			"Finding spot price for %s %s...",
			s.SpotPriceProduct, s.InstanceType))

		// Detect the spot price
		startTime := time.Now().Add(-1 * time.Hour)
		resp, err := ec2conn.DescribeSpotPriceHistory(&ec2.DescribeSpotPriceHistory{
			InstanceType:       []string{s.RunConfig.InstanceType},
			ProductDescription: []string{s.RunConfig.SpotPriceAutoProduct},
			AvailabilityZone:   s.RunConfig.AvailabilityZone,
			StartTime:          startTime,
		})
		if err != nil {
			err := fmt.Errorf("Error finding spot price: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		var price float64
		for _, history := range resp.History {
			log.Printf("[INFO] Candidate spot price: %s", history.SpotPrice)
			current, err := strconv.ParseFloat(history.SpotPrice, 64)
			if err != nil {
				log.Printf("[ERR] Error parsing spot price: %s", err)
				continue
			}
			if price == 0 || current < price {
				price = current
			}
		}
		if price == 0 {
			err := fmt.Errorf("No candidate spot prices found!")
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		spotPrice = strconv.FormatFloat(price, 'f', -1, 64)
	}

	var instanceId string

	if spotPrice == "" {
		runOpts := &ec2.RunInstances{
			KeyName:                  keyName,
			ImageId:                  s.RunConfig.SourceAmi,
			InstanceType:             s.RunConfig.InstanceType,
			UserData:                 []byte(userData),
			MinCount:                 0,
			MaxCount:                 0,
			SecurityGroups:           securityGroups,
			IamInstanceProfile:       s.RunConfig.IamInstanceProfile,
			SubnetId:                 s.RunConfig.SubnetId,
			AssociatePublicIpAddress: s.RunConfig.AssociatePublicIpAddress,
			BlockDevices:             s.BlockDevices.BuildLaunchDevices(),
			AvailZone:                s.RunConfig.AvailabilityZone,
		}
		runResp, err := ec2conn.RunInstances(runOpts)
		if err != nil {
			err := fmt.Errorf("Error launching source instance: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
		instanceId = runResp.Instances[0].InstanceId
	} else {
		ui.Message(fmt.Sprintf(
			"Requesting spot instance '%s' for: %s",
			s.RunConfig.InstanceType, spotPrice))

		runOpts := &ec2.RequestSpotInstances{
			SpotPrice:                spotPrice,
			KeyName:                  keyName,
			ImageId:                  s.RunConfig.SourceAmi,
			InstanceType:             s.RunConfig.InstanceType,
			UserData:                 []byte(userData),
			SecurityGroups:           securityGroups,
			IamInstanceProfile:       s.RunConfig.IamInstanceProfile,
			SubnetId:                 s.RunConfig.SubnetId,
			AssociatePublicIpAddress: s.RunConfig.AssociatePublicIpAddress,
			BlockDevices:             s.BlockDevices.BuildLaunchDevices(),
			AvailZone:                s.RunConfig.AvailabilityZone,
		}
		runSpotResp, err := ec2conn.RequestSpotInstances(runOpts)
		if err != nil {
			err := fmt.Errorf("Error launching source spot instance: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		s.spotRequest = &runSpotResp.SpotRequestResults[0]

		spotRequestId := s.spotRequest.SpotRequestId
		ui.Message(fmt.Sprintf("Waiting for spot request (%s) to become active...", spotRequestId))
		stateChange := awscommon.StateChangeConf{
			Pending:   []string{"open"},
			Target:    "active",
			Refresh:   awscommon.SpotRequestStateRefreshFunc(ec2conn, spotRequestId),
			StepState: state,
		}
		_, err = awscommon.WaitForState(&stateChange)
		if err != nil {
			err := fmt.Errorf("Error waiting for spot request (%s) to become ready: %s", spotRequestId, err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
		spotResp, err := ec2conn.DescribeSpotRequests([]string{spotRequestId}, nil)
		if err != nil {
			err := fmt.Errorf("Error finding spot request (%s): %s", spotRequestId, err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
		instanceId = spotResp.SpotRequestResults[0].InstanceId
	}

	instanceResp, err := ec2conn.Instances([]string{instanceId}, nil)
	if err != nil {
		err := fmt.Errorf("Error finding source instance (%s): %s", instanceId, err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}
	s.instance = &instanceResp.Reservations[0].Instances[0]
	ui.Message(fmt.Sprintf("Instance ID: %s", s.instance.InstanceId))

	ui.Say(fmt.Sprintf("Waiting for instance (%s) to become ready...", s.instance.InstanceId))
	stateChange := awscommon.StateChangeConf{
		Pending:   []string{"pending"},
		Target:    "running",
		Refresh:   awscommon.InstanceStateRefreshFunc(ec2conn, s.instance),
		StepState: state,
	}
	latestInstance, err := awscommon.WaitForState(&stateChange)
	if err != nil {
		err := fmt.Errorf("Error waiting for instance (%s) to become ready: %s", s.instance.InstanceId, err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	s.instance = latestInstance.(*ec2.Instance)

	ec2Tags := make([]ec2.Tag, 1, len(s.RunConfig.RunTags)+1)
	ec2Tags[0] = ec2.Tag{"Name", "Packer Builder"}
	for k, v := range s.RunConfig.RunTags {
		ec2Tags = append(ec2Tags, ec2.Tag{k, v})
	}

	_, err = ec2conn.CreateTags([]string{s.instance.InstanceId}, ec2Tags)
	if err != nil {
		ui.Message(
			fmt.Sprintf("Failed to tag a Name on the builder instance: %s", err))
	}

	if s.Debug {
		if s.instance.DNSName != "" {
			ui.Message(fmt.Sprintf("Public DNS: %s", s.instance.DNSName))
		}

		if s.instance.PublicIpAddress != "" {
			ui.Message(fmt.Sprintf("Public IP: %s", s.instance.PublicIpAddress))
		}

		if s.instance.PrivateIpAddress != "" {
			ui.Message(fmt.Sprintf("Private IP: %s", s.instance.PrivateIpAddress))
		}
	}

	state.Put("instance", s.instance)

	return multistep.ActionContinue
}

func (s *StepRunSourceInstance) Cleanup(state multistep.StateBag) {

	ec2conn := state.Get("ec2").(*ec2.EC2)
	ui := state.Get("ui").(packer.Ui)

	// Cancel the spot request if it exists
	if s.spotRequest != nil {
		ui.Say("Cancelling the spot request...")
		if _, err := ec2conn.CancelSpotRequests([]string{s.spotRequest.SpotRequestId}); err != nil {
			ui.Error(fmt.Sprintf("Error cancelling the spot request, may still be around: %s", err))
			return
		}
		stateChange := awscommon.StateChangeConf{
			Pending: []string{"active", "open"},
			Refresh: awscommon.SpotRequestStateRefreshFunc(ec2conn, s.spotRequest.SpotRequestId),
			Target:  "cancelled",
		}

		awscommon.WaitForState(&stateChange)

	}

	// Terminate the source instance if it exists
	if s.instance != nil {

		ui.Say("Terminating the source AWS instance...")
		if _, err := ec2conn.TerminateInstances([]string{s.instance.InstanceId}); err != nil {
			ui.Error(fmt.Sprintf("Error terminating instance, may still be around: %s", err))
			return
		}
		stateChange := awscommon.StateChangeConf{
			Pending: []string{"pending", "running", "shutting-down", "stopped", "stopping"},
			Refresh: awscommon.InstanceStateRefreshFunc(ec2conn, s.instance),
			Target:  "terminated",
		}

		awscommon.WaitForState(&stateChange)
	}
}
