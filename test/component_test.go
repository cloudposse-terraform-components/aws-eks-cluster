package test

import (
	"context"
	"testing"
	"fmt"
	"strings"
	helper "github.com/cloudposse/test-helpers/pkg/atmos/component-helper"
	awsHelper "github.com/cloudposse/test-helpers/pkg/aws"
	"github.com/cloudposse/test-helpers/pkg/atmos"
	"github.com/gruntwork-io/terratest/modules/aws"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ComponentSuite struct {
	helper.TestSuite
}

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
	assert.True(s.T(), strings.HasSuffix(endpoint, fmt.Sprintf("%s.eks.amazonaws.com", awsRegion)))

	oidcIssuer := atmos.Output(s.T(), options, "eks_cluster_identity_oidc_issuer")
	assert.True(s.T(), strings.HasPrefix(oidcIssuer, fmt.Sprintf("https://oidc.eks.%s.amazonaws.com/id", awsRegion)))

	certificateAuthorityData := atmos.Output(s.T(), options, "eks_cluster_certificate_authority_data")
	assert.NotNil(s.T(), certificateAuthorityData)

	managedSecurityGroupId := atmos.Output(s.T(), options, "eks_cluster_managed_security_group_id")
	assert.True(s.T(), strings.HasPrefix(managedSecurityGroupId, "sg-"))

	version := atmos.Output(s.T(), options, "eks_cluster_version")
	assert.Equal(s.T(), version, "1.30")

	nodeGroupArns := atmos.OutputList(s.T(), options, "eks_node_group_arns")
	assert.Equal(s.T(), len(nodeGroupArns), 2)

	managedNodeWorkersRoleArns := atmos.OutputList(s.T(), options, "eks_managed_node_workers_role_arns")
	assert.Equal(s.T(), len(managedNodeWorkersRoleArns), 2)

	nodeGroupCount := atmos.Output(s.T(), options, "eks_node_group_count")
	assert.Equal(s.T(), nodeGroupCount, "2")

	nodeGroupIds := atmos.OutputList(s.T(), options, "eks_node_group_ids")
	assert.Equal(s.T(), len(nodeGroupIds), 2)

	nodeGroupRoleNames := atmos.OutputList(s.T(), options, "eks_node_group_role_names")
	assert.Equal(s.T(), len(nodeGroupRoleNames), 2)

	authWorkerRoles := atmos.OutputList(s.T(), options, "eks_auth_worker_roles")
	assert.Equal(s.T(), len(authWorkerRoles), 1)

	nodeGroupStatuses := atmos.OutputList(s.T(), options, "eks_node_group_statuses")
	assert.Equal(s.T(), len(nodeGroupStatuses), 2)
	assert.Equal(s.T(), nodeGroupStatuses[0], "ACTIVE")
	assert.Equal(s.T(), nodeGroupStatuses[1], "ACTIVE")

	karpenterIamRoleName := atmos.Output(s.T(), options, "karpenter_iam_role_name")
	assert.True(s.T(), strings.HasPrefix(karpenterIamRoleName, "eg-default-ue2-test-"))

	karpenterIamRoleArn := atmos.Output(s.T(), options, "karpenter_iam_role_arn")
	assert.True(s.T(), strings.HasPrefix(karpenterIamRoleArn, fmt.Sprintf("arn:aws:iam::%s:role/%s", accountId, karpenterIamRoleName)))

	fargateProfileRoleNames := atmos.OutputList(s.T(), options, "fargate_profile_role_names")
	assert.Equal(s.T(), len(fargateProfileRoleNames), 1)
	assert.True(s.T(), strings.HasPrefix(fargateProfileRoleNames[0], "eg-default-ue2-test-"))

	fargateProfileRoleArns := atmos.OutputList(s.T(), options, "fargate_profile_role_arns")
	assert.Equal(s.T(), len(fargateProfileRoleArns), 1)
	assert.True(s.T(), strings.HasPrefix(fargateProfileRoleArns[0], fmt.Sprintf("arn:aws:iam::%s:role/%s", accountId, fargateProfileRoleNames[0])))

	fargateProfiles := atmos.Output(s.T(), options, "fargate_profiles")
	assert.NotNil(s.T(), fargateProfiles)
	// {
	// 	"karpenter": {
	// 		"eks_fargate_pod_execution_role_arn": "arn:aws:iam::799847381734:role/eg-default-ue2-test-p76hvz-cluster-fargate",
	// 		"eks_fargate_pod_execution_role_name":"eg-default-ue2-test-p76hvz-cluster-fargate",
	// 		"eks_fargate_profile_arn":"arn:aws:eks:us-east-2:799847381734:fargateprofile/eg-default-ue2-test-p76hvz-cluster/eg-default-ue2-test-p76hvz-karpenter/7ccaca54-e2f7-b754-c425-e7f02cf10d09",
	// 		"eks_fargate_profile_id":"eg-default-ue2-test-p76hvz-cluster:eg-default-ue2-test-p76hvz-karpenter",
	// 		"eks_fargate_profile_role_arn":"arn:aws:iam::799847381734:role/eg-default-ue2-test-p76hvz-cluster-fargate",
	// 		"eks_fargate_profile_role_name":"eg-default-ue2-test-p76hvz-cluster-fargate",
	// 		"eks_fargate_profile_status":"ACTIVE"
	// 	}
	// }

	vpcCidr := atmos.Output(s.T(), options, "vpc_cidr")
	assert.Equal(s.T(), vpcCidr, "172.16.0.0/16")

	availabilityZones := atmos.OutputList(s.T(), options, "availability_zones")
	assert.Equal(s.T(), len(availabilityZones), 2)

	eksAddonsVersions := atmos.OutputMapOfObjects(s.T(), options, "eks_addons_versions")
	assert.NotNil(s.T(), eksAddonsVersions)
	assert.Equal(s.T(), len(eksAddonsVersions), 5)
	assert.Equal(s.T(), eksAddonsVersions["aws-ebs-csi-driver"], "v1.34.0-eksbuild.1")
	assert.Equal(s.T(), eksAddonsVersions["aws-efs-csi-driver"], "v2.0.8-eksbuild.1")
	assert.Equal(s.T(), eksAddonsVersions["coredns"], "v1.11.3-eksbuild.1")
	assert.Equal(s.T(), eksAddonsVersions["kube-proxy"], "v1.30.3-eksbuild.5")
	assert.Equal(s.T(), eksAddonsVersions["vpc-cni"], "v1.18.3-eksbuild.3")

	cluster := awsHelper.GetEksCluster(s.T(), context.Background(), awsRegion, id)
	assert.Equal(s.T(), *cluster.Name, id)
	assert.Equal(s.T(), *cluster.Arn, arn)
	assert.Equal(s.T(), *cluster.Endpoint, endpoint)
	assert.Equal(s.T(), *cluster.Identity.Oidc.Issuer, oidcIssuer)
	assert.Equal(s.T(), string(cluster.Status), "ACTIVE")
	assert.Equal(s.T(), *cluster.Version, "1.30")

	clientset, err := awsHelper.NewK8SClientset(cluster)
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), clientset)

	namespaces, err := clientset.CoreV1().Namespaces().List(context.Background(), metav1.ListOptions{})
	assert.NoError(s.T(), err)
	assert.NotNil(s.T(), namespaces)
	assert.Equal(s.T(), len(namespaces.Items), 4)

	s.DriftTest(component, stack, nil)
}

func (s *ComponentSuite) TestEnabledFlag() {
	const component = "eks/cluster/disabled"
	const stack = "default-test"
	s.VerifyEnabledFlag(component, stack, nil)
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
