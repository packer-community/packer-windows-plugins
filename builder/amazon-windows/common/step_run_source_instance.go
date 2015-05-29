package common

import (
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"time"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/ec2"
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
	spotRequest *ec2.SpotInstanceRequest
}

func (s *StepRunSourceInstance) Run(state multistep.StateBag) multistep.StepAction {
	ec2conn := state.Get("ec2").(*ec2.EC2)
	keyName := state.Get("keyPair").(string)
	tempSecurityGroupIds := state.Get("securityGroupIds").([]string)
	ui := state.Get("ui").(packer.Ui)

	securityGroupIds := make([]*string, len(tempSecurityGroupIds))
	for i, sg := range tempSecurityGroupIds {
		securityGroupIds[i] = &sg
	}

	userData := s.RunConfig.UserData
	if s.RunConfig.UserDataFile != "" {
		contents, err := ioutil.ReadFile(s.RunConfig.UserDataFile)
		if err != nil {
			state.Put("error", fmt.Errorf("Problem reading user data file: %s", err))
			return multistep.ActionHalt
		}

		userData = string(contents)
	}

	ui.Say("Launching a source AWS instance...")
	imageResp, err := ec2conn.DescribeImages(&ec2.DescribeImagesInput{
		ImageIDs: []*string{&s.RunConfig.SourceAmi},
	})
	if err != nil {
		state.Put("error", fmt.Errorf("There was a problem with the source AMI: %s", err))
		return multistep.ActionHalt
	}

	if len(imageResp.Images) != 1 {
		state.Put("error", fmt.Errorf("The source AMI '%s' could not be found.", s.RunConfig.SourceAmi))
		return multistep.ActionHalt
	}

	if s.ExpectedRootDevice != "" && *imageResp.Images[0].RootDeviceType != s.ExpectedRootDevice {
		state.Put("error", fmt.Errorf(
			"The provided source AMI has an invalid root device type.\n"+
				"Expected '%s', got '%s'.",
			s.ExpectedRootDevice, *imageResp.Images[0].RootDeviceType))
		return multistep.ActionHalt
	}

	spotPrice := s.RunConfig.SpotPrice
	if spotPrice == "auto" {
		ui.Message(fmt.Sprintf(
			"Finding spot price for %s %s...",
			s.RunConfig.SpotPriceAutoProduct, s.RunConfig.InstanceType))

		// Detect the spot price
		startTime := time.Now().Add(-1 * time.Hour)
		resp, err := ec2conn.DescribeSpotPriceHistory(&ec2.DescribeSpotPriceHistoryInput{
			InstanceTypes:       []*string{&s.RunConfig.InstanceType},
			ProductDescriptions: []*string{&s.RunConfig.SpotPriceAutoProduct},
			AvailabilityZone:    &s.RunConfig.AvailabilityZone,
			StartTime:           &startTime,
		})
		if err != nil {
			err := fmt.Errorf("Error finding spot price: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		var price float64
		for _, history := range resp.SpotPriceHistory {
			log.Printf("[INFO] Candidate spot price: %s", history.SpotPrice)
			current, err := strconv.ParseFloat(*history.SpotPrice, 64)
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
		runOpts := &ec2.RunInstancesInput{
			KeyName:             &keyName,
			ImageID:             &s.RunConfig.SourceAmi,
			InstanceType:        &s.RunConfig.InstanceType,
			UserData:            &userData,
			MaxCount:            aws.Long(1),
			MinCount:            aws.Long(1),
			IAMInstanceProfile:  &ec2.IAMInstanceProfileSpecification{Name: &s.RunConfig.IamInstanceProfile},
			BlockDeviceMappings: s.BlockDevices.BuildLaunchDevices(),
			Placement:           &ec2.Placement{AvailabilityZone: &s.RunConfig.AvailabilityZone},
		}

		if s.RunConfig.SubnetId != "" && s.RunConfig.AssociatePublicIpAddress {
			runOpts.NetworkInterfaces = []*ec2.InstanceNetworkInterfaceSpecification{
				&ec2.InstanceNetworkInterfaceSpecification{
					DeviceIndex:              aws.Long(0),
					AssociatePublicIPAddress: &s.RunConfig.AssociatePublicIpAddress,
					SubnetID:                 &s.RunConfig.SubnetId,
					Groups:                   securityGroupIds,
					DeleteOnTermination:      aws.Boolean(true),
				},
			}
		} else {
			runOpts.SubnetID = &s.RunConfig.SubnetId
			runOpts.SecurityGroupIDs = securityGroupIds
		}

		runResp, err := ec2conn.RunInstances(runOpts)
		if err != nil {
			err := fmt.Errorf("Error launching source instance: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
		instanceId = *runResp.Instances[0].InstanceID
	} else {
		ui.Message(fmt.Sprintf(
			"Requesting spot instance '%s' for: %s",
			s.RunConfig.InstanceType, spotPrice))

		runSpotResp, err := ec2conn.RequestSpotInstances(&ec2.RequestSpotInstancesInput{
			SpotPrice: &spotPrice,
			LaunchSpecification: &ec2.RequestSpotLaunchSpecification{
				KeyName:            &keyName,
				ImageID:            &s.RunConfig.SourceAmi,
				InstanceType:       &s.RunConfig.InstanceType,
				UserData:           &userData,
				SecurityGroupIDs:   securityGroupIds,
				IAMInstanceProfile: &ec2.IAMInstanceProfileSpecification{Name: &s.RunConfig.IamInstanceProfile},
				SubnetID:           &s.RunConfig.SubnetId,
				NetworkInterfaces: []*ec2.InstanceNetworkInterfaceSpecification{
					&ec2.InstanceNetworkInterfaceSpecification{AssociatePublicIPAddress: &s.RunConfig.AssociatePublicIpAddress},
				},
				Placement:           &ec2.SpotPlacement{AvailabilityZone: &s.RunConfig.AvailabilityZone},
				BlockDeviceMappings: s.BlockDevices.BuildLaunchDevices(),
			},
		})
		if err != nil {
			err := fmt.Errorf("Error launching source spot instance: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		s.spotRequest = runSpotResp.SpotInstanceRequests[0]

		spotRequestId := s.spotRequest.SpotInstanceRequestID
		ui.Message(fmt.Sprintf("Waiting for spot request (%s) to become active...", spotRequestId))
		stateChange := awscommon.StateChangeConf{
			Pending:   []string{"open"},
			Target:    "active",
			Refresh:   awscommon.SpotRequestStateRefreshFunc(ec2conn, *spotRequestId),
			StepState: state,
		}
		_, err = awscommon.WaitForState(&stateChange)
		if err != nil {
			err := fmt.Errorf("Error waiting for spot request (%s) to become ready: %s", spotRequestId, err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
		spotResp, err := ec2conn.DescribeSpotInstanceRequests(&ec2.DescribeSpotInstanceRequestsInput{
			SpotInstanceRequestIDs: []*string{spotRequestId},
		})
		if err != nil {
			err := fmt.Errorf("Error finding spot request (%s): %s", spotRequestId, err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
		instanceId = *spotResp.SpotInstanceRequests[0].InstanceID
	}

	instanceResp, err := ec2conn.DescribeInstances(&ec2.DescribeInstancesInput{InstanceIDs: []*string{&instanceId}})
	if err != nil {
		err := fmt.Errorf("Error finding source instance (%s): %s", instanceId, err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}
	s.instance = instanceResp.Reservations[0].Instances[0]
	ui.Message(fmt.Sprintf("Instance ID: %s", *s.instance.InstanceID))

	ui.Say(fmt.Sprintf("Waiting for instance (%s) to become ready...", *s.instance.InstanceID))
	stateChange := awscommon.StateChangeConf{
		Pending:   []string{"pending"},
		Target:    "running",
		Refresh:   awscommon.InstanceStateRefreshFunc(ec2conn, s.instance),
		StepState: state,
	}
	latestInstance, err := awscommon.WaitForState(&stateChange)
	if err != nil {
		err := fmt.Errorf("Error waiting for instance (%s) to become ready: %s", *s.instance.InstanceID, err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	s.instance = latestInstance.(*ec2.Instance)

	ec2Tags := make([]*ec2.Tag, 1, len(s.RunConfig.RunTags)+1)
	ec2Tags[0] = &ec2.Tag{Key: aws.String("Name"), Value: aws.String("Packer Builder")}
	for k, v := range s.RunConfig.RunTags {
		ec2Tags = append(ec2Tags, &ec2.Tag{Key: &k, Value: &v})
	}

	_, err = ec2conn.CreateTags(&ec2.CreateTagsInput{
		Tags:      ec2Tags,
		Resources: []*string{s.instance.InstanceID},
	})
	if err != nil {
		ui.Message(
			fmt.Sprintf("Failed to tag a Name on the builder instance: %s", err))
	}

	if s.Debug {
		if s.instance.PublicDNSName != nil && *s.instance.PublicDNSName != "" {
			ui.Message(fmt.Sprintf("Public DNS: %s", *s.instance.PublicDNSName))
		}

		if s.instance.PublicIPAddress != nil && *s.instance.PublicIPAddress != "" {
			ui.Message(fmt.Sprintf("Public IP: %s", *s.instance.PublicIPAddress))
		}

		if s.instance.PrivateIPAddress != nil && *s.instance.PrivateIPAddress != "" {
			ui.Message(fmt.Sprintf("Private IP: %s", *s.instance.PrivateIPAddress))
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
		input := &ec2.CancelSpotInstanceRequestsInput{
			SpotInstanceRequestIDs: []*string{s.spotRequest.InstanceID},
		}
		if _, err := ec2conn.CancelSpotInstanceRequests(input); err != nil {
			ui.Error(fmt.Sprintf("Error cancelling the spot request, may still be around: %s", err))
			return
		}
		stateChange := StateChangeConf{
			Pending: []string{"active", "open"},
			Refresh: SpotRequestStateRefreshFunc(ec2conn, *s.spotRequest.SpotInstanceRequestID),
			Target:  "cancelled",
		}

		WaitForState(&stateChange)

	}

	// Terminate the source instance if it exists
	if s.instance != nil {

		ui.Say("Terminating the source AWS instance...")
		if _, err := ec2conn.TerminateInstances(&ec2.TerminateInstancesInput{InstanceIDs: []*string{s.instance.InstanceID}}); err != nil {
			ui.Error(fmt.Sprintf("Error terminating instance, may still be around: %s", err))
			return
		}
		stateChange := StateChangeConf{
			Pending: []string{"pending", "running", "shutting-down", "stopped", "stopping"},
			Refresh: InstanceStateRefreshFunc(ec2conn, s.instance),
			Target:  "terminated",
		}

		WaitForState(&stateChange)
	}
}
