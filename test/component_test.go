package test

import (
	"testing"
	"fmt"
	"strings"
	helper "github.com/cloudposse/test-helpers/pkg/atmos/component-helper"
	"github.com/cloudposse/test-helpers/pkg/atmos"
	"github.com/gruntwork-io/terratest/modules/aws"
	"github.com/stretchr/testify/assert"
)

type ComponentSuite struct {
	helper.TestSuite
}

// eks/cluster



func (s *ComponentSuite) TestBasic() {
	const component = "eks/cluster/basic"
	const stack = "default-test"
	const awsRegion = "us-east-2"

	defer s.DestroyAtmosComponent(s.T(), component, stack, nil)
	options, _ := s.DeployAtmosComponent(s.T(), component, stack, nil)
	assert.NotNil(s.T(), options)

	accountId := aws.GetAccountId(s.T())
	assert.NotNil(s.T(), accountId)

	id := atmos.Output(s.T(), options, "eks_cluster_id")
	assert.True(s.T(), strings.HasPrefix(id, "eg-default-ue2-test-"))

	arn := atmos.Output(s.T(), options, "eks_cluster_arn")
	assert.Equal(s.T(), arn, fmt.Sprintf("arn:aws:eks:%s:%s:cluster/%s", awsRegion, accountId, id))

	endpoint := atmos.Output(s.T(), options, "eks_cluster_endpoint")
	assert.NotNil(s.T(), endpoint)

	oidcIssuer := atmos.Output(s.T(), options, "eks_cluster_identity_oidc_issuer")
	assert.NotNil(s.T(), oidcIssuer)

	certificateAuthorityData := atmos.Output(s.T(), options, "eks_cluster_certificate_authority_data")
	assert.NotNil(s.T(), certificateAuthorityData)

	managedSecurityGroupId := atmos.Output(s.T(), options, "eks_cluster_managed_security_group_id")
	assert.NotNil(s.T(), managedSecurityGroupId)

	version := atmos.Output(s.T(), options, "eks_cluster_version")
	assert.NotNil(s.T(), version)

	nodeGroupArns := atmos.Output(s.T(), options, "eks_node_group_arns")
	assert.NotNil(s.T(), nodeGroupArns)

	managedNodeWorkersRoleArns := atmos.Output(s.T(), options, "eks_managed_node_workers_role_arns")
	assert.NotNil(s.T(), managedNodeWorkersRoleArns)

	nodeGroupCount := atmos.Output(s.T(), options, "eks_node_group_count")
	assert.NotNil(s.T(), nodeGroupCount)

	nodeGroupIds := atmos.Output(s.T(), options, "eks_node_group_ids")
	assert.NotNil(s.T(), nodeGroupIds)

	nodeGroupRoleNames := atmos.Output(s.T(), options, "eks_node_group_role_names")
	assert.NotNil(s.T(), nodeGroupRoleNames)

	authWorkerRoles := atmos.Output(s.T(), options, "eks_auth_worker_roles")
	assert.NotNil(s.T(), authWorkerRoles)

	nodeGroupStatuses := atmos.Output(s.T(), options, "eks_node_group_statuses")
	assert.NotNil(s.T(), nodeGroupStatuses)

	karpenterIamRoleArn := atmos.Output(s.T(), options, "karpenter_iam_role_arn")
	assert.NotNil(s.T(), karpenterIamRoleArn)

	karpenterIamRoleName := atmos.Output(s.T(), options, "karpenter_iam_role_name")
	assert.NotNil(s.T(), karpenterIamRoleName)

	fargateProfiles := atmos.Output(s.T(), options, "fargate_profiles")
	assert.NotNil(s.T(), fargateProfiles)

	fargateProfileRoleArns := atmos.Output(s.T(), options, "fargate_profile_role_arns")
	assert.NotNil(s.T(), fargateProfileRoleArns)

	fargateProfileRoleNames := atmos.Output(s.T(), options, "fargate_profile_role_names")
	assert.NotNil(s.T(), fargateProfileRoleNames)

	vpcCidr := atmos.Output(s.T(), options, "vpc_cidr")
	assert.NotNil(s.T(), vpcCidr)

	availabilityZones := atmos.Output(s.T(), options, "availability_zones")
	assert.NotNil(s.T(), availabilityZones)

	eksAddonsVersions := atmos.Output(s.T(), options, "eks_addons_versions")
	assert.NotNil(s.T(), eksAddonsVersions)


	// s.DriftTest(component, stack, nil)
}

func (s *ComponentSuite) TestEnabledFlag() {
	// const component = "example/disabled"
	// const stack = "default-test"
	// s.VerifyEnabledFlag(component, stack, nil)
}

func (s *ComponentSuite) SetupSuite() {
	s.TestSuite.InitConfig()
	s.TestSuite.Config.ComponentDestDir = "components/terraform/eks/cluster"
	s.TestSuite.SetupSuite()
}

func TestRunSuite(t *testing.T) {
	suite := new(ComponentSuite)
	suite.AddDependency(t, "vpc", "default-test", nil)
	helper.Run(t, suite)
}
