package main

import (
	"os"

	"github.com/aws/aws-cdk-go/awscdk/v2"

	"github.com/aws/aws-cdk-go/awscdk/v2/awsec2"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsecs"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsecspatterns"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsiam"
	"github.com/aws/aws-cdk-go/awscdk/v2/awsservicediscovery"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
)

type CdkGoPlaygroundStackProps struct {
	awscdk.StackProps
}

func NewCdkGoPlaygroundStack(scope constructs.Construct, id string, props *CdkGoPlaygroundStackProps) awscdk.Stack {
	var sprops awscdk.StackProps
	if props != nil {
		sprops = props.StackProps
	}
	stack := awscdk.NewStack(scope, &id, &sprops)

	vpc := awsec2.Vpc_FromLookup(stack, jsii.String("testVPC"), &awsec2.VpcLookupOptions{
		VpcName: jsii.String("DemoGoECSVPC"),
	})

	cluster := awsecs.NewCluster(stack, jsii.String("adServicesCluster"), &awsecs.ClusterProps{
		ClusterName: jsii.String("ad-services-cluster"),
		DefaultCloudMapNamespace: &awsecs.CloudMapNamespaceOptions{
			Name: jsii.String("service.local"),
		},
		Vpc: vpc,
	})

	// Using constract NewApplicationLoadBalancedFargateService
	loadBalancedFargateService := awsecspatterns.NewApplicationLoadBalancedFargateService(stack, jsii.String("adService"), &awsecspatterns.ApplicationLoadBalancedFargateServiceProps{
		Cluster:        cluster,
		MemoryLimitMiB: jsii.Number(1024),
		Cpu:            jsii.Number(512),
		TaskImageOptions: &awsecspatterns.ApplicationLoadBalancedTaskImageOptions{
			Image: awsecs.ContainerImage_FromRegistry(jsii.String("<Account_ID>.dkr.ecr.us-east-1.amazonaws.com/ad-server:0.0.1"), &awsecs.RepositoryImageProps{}),
		},
		AssignPublicIp:   jsii.Bool(false),
		LoadBalancerName: jsii.String("DemoLoadBalancer"),
		ServiceName:      jsii.String("ad-service"),
		CloudMapOptions: &awsecs.CloudMapOptions{
			// Create A records - useful for AWSVPC network mode.
			DnsRecordType: awsservicediscovery.DnsRecordType_A,
			// The name of the Cloud Map service to attach to the ECS service.
			Name: jsii.String("adservice"),
		},
		TaskSubnets: &awsec2.SubnetSelection{
			SubnetType: awsec2.SubnetType_PRIVATE_WITH_NAT,
		},
	})

	// Using Amazon ECR images with Amazon ECS
	// https://docs.aws.amazon.com/AmazonECR/latest/userguide/ECR_on_ECS.html
	//https://docs.aws.amazon.com/AmazonECS/latest/bestpracticesguide/security-iam-roles.html
	loadBalancedFargateService.TaskDefinition().AddToExecutionRolePolicy(
		awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
			Effect: awsiam.Effect_ALLOW,
			Actions: &[]*string{
				jsii.String("ecr:BatchGetImage"),
				jsii.String("ecr:GetDownloadUrlForLayer"),
				jsii.String("ecr:GetAuthorizationToken"),
			},
			Resources: &[]*string{
				jsii.String("*"),
			},
		}))

	tdRecomendationApp := awsecs.NewFargateTaskDefinition(stack, jsii.String("RecomendationAppECSTaskDefinition"), &awsecs.FargateTaskDefinitionProps{
		MemoryLimitMiB: jsii.Number(512),
		Cpu:            jsii.Number(256),
	})

	tdRecomendationApp.AddContainer(jsii.String("RecomendationAppContainer"), &awsecs.ContainerDefinitionOptions{
		Image:          awsecs.ContainerImage_FromRegistry(jsii.String("<Account_ID>.dkr.ecr.us-east-1.amazonaws.com/ad-recommender:0.0.1"), &awsecs.RepositoryImageProps{}),
		MemoryLimitMiB: jsii.Number(256),
		Logging: awsecs.LogDrivers_AwsLogs(&awsecs.AwsLogDriverProps{
			StreamPrefix: jsii.String("recommendation-app"),
		}),
	})

	tdRecomendationApp.AddToExecutionRolePolicy(
		awsiam.NewPolicyStatement(&awsiam.PolicyStatementProps{
			Effect: awsiam.Effect_ALLOW,
			Actions: &[]*string{
				jsii.String("ecr:BatchGetImage"),
				jsii.String("ecr:GetDownloadUrlForLayer"),
				jsii.String("ecr:GetAuthorizationToken"),
			},
			Resources: &[]*string{
				jsii.String("*"),
			},
		}),
	)

	recomendationService := awsecs.NewFargateService(stack, jsii.String("recomendationAppECSService"), &awsecs.FargateServiceProps{
		Cluster:        cluster,
		TaskDefinition: tdRecomendationApp,
		DesiredCount:   jsii.Number(1),
		ServiceName:    jsii.String("ad-recommender"),
		CloudMapOptions: &awsecs.CloudMapOptions{
			// Create A records - useful for AWSVPC network mode.
			DnsRecordType: awsservicediscovery.DnsRecordType_A,
			// The name of the Cloud Map service to attach to the ECS service.
			Name: jsii.String("recommendation"),
		},
		VpcSubnets: &awsec2.SubnetSelection{
			SubnetType: awsec2.SubnetType_PRIVATE_ISOLATED,
		},
	})

	_ = recomendationService

	return stack
}

func main() {
	defer jsii.Close()

	app := awscdk.NewApp(nil)

	NewCdkGoPlaygroundStack(app, "AdServices", &CdkGoPlaygroundStackProps{
		awscdk.StackProps{
			Env: env(),
		},
	})

	app.Synth(nil)
}

// env determines the AWS environment (account+region) in which our stack is to
// be deployed. For more information see: https://docs.aws.amazon.com/cdk/latest/guide/environments.html
func env() *awscdk.Environment {
	return &awscdk.Environment{
		Account: jsii.String(os.Getenv("CDK_DEFAULT_ACCOUNT")),
		Region:  jsii.String(os.Getenv("CDK_DEFAULT_REGION")),
	}
}
