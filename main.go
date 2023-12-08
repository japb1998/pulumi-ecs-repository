package main

import (
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/ecs"
	"github.com/pulumi/pulumi-aws/sdk/v6/go/aws/lb"
	"github.com/pulumi/pulumi-awsx/sdk/v2/go/awsx/ecr"
	ecrx "github.com/pulumi/pulumi-awsx/sdk/v2/go/awsx/ecr"
	ecsx "github.com/pulumi/pulumi-awsx/sdk/v2/go/awsx/ecs"
	lbx "github.com/pulumi/pulumi-awsx/sdk/v2/go/awsx/lb"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi/config"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		cfg := config.New(ctx, "")

		// VPC for my app
		vpc, err := ec2.NewVpc(ctx, "vpc", &ec2.VpcArgs{
			CidrBlock: pulumi.String("10.0.0.0/16"),
		})

		if err != nil {
			return err
		}

		// internet gateway necessary for internet connection within the vpc
		ig, err := ec2.NewInternetGateway(ctx, "igw", &ec2.InternetGatewayArgs{
			VpcId: vpc.ID(),
			Tags: pulumi.StringMap{
				"Name": pulumi.String("Control-tower"),
			},
		})
		if err != nil {
			return err
		}

		// Private
		eIp, err := ec2.NewEip(ctx, "elasticIp", &ec2.EipArgs{
			Domain: pulumi.String("vpc"),
		})
		if err != nil {
			return err
		}

		// Private
		eIpTwo, err := ec2.NewEip(ctx, "elasticIpSecond", &ec2.EipArgs{
			Domain: pulumi.String("vpc"),
		})
		if err != nil {
			return err
		}

		//private subnets
		privateSubnetOne, err := ec2.NewSubnet(ctx, "privateSubnetOne", &ec2.SubnetArgs{
			VpcId:     vpc.ID(),
			CidrBlock: pulumi.String("10.0.3.0/24"),
			Tags: pulumi.StringMap{
				"Name": pulumi.String("Main"),
			},
			AvailabilityZone: pulumi.String("us-east-1a"),
		})

		if err != nil {
			return nil
		}

		privateSubnetTwo, err := ec2.NewSubnet(ctx, "privateSubnetTwo", &ec2.SubnetArgs{
			VpcId:     vpc.ID(),
			CidrBlock: pulumi.String("10.0.4.0/24"),
			Tags: pulumi.StringMap{
				"Name": pulumi.String("Main"),
			},
			AvailabilityZone: pulumi.String("us-east-1b"),
		})

		if err != nil {
			return nil
		}
		// PUBLIC SUBNETS //
		subOne, err := ec2.NewSubnet(ctx, "publicSubnet", &ec2.SubnetArgs{
			VpcId:     vpc.ID(),
			CidrBlock: pulumi.String("10.0.1.0/24"),
			Tags: pulumi.StringMap{
				"Name": pulumi.String("Main"),
			},
			AvailabilityZone:    pulumi.String("us-east-1a"),
			MapPublicIpOnLaunch: pulumi.Bool(true),
		})

		if err != nil {
			return nil
		}

		subTwo, err := ec2.NewSubnet(ctx, "publicTwo", &ec2.SubnetArgs{
			VpcId:     vpc.ID().ToStringOutput(),
			CidrBlock: pulumi.String("10.0.2.0/24"),
			Tags: pulumi.StringMap{
				"Name": pulumi.String("Main"),
			},
			AvailabilityZone:    pulumi.String("us-east-1b"),
			MapPublicIpOnLaunch: pulumi.Bool(true),
		})

		if err != nil {
			return nil
		}
		// Ends here //
		// NAT
		nat, err := ec2.NewNatGateway(ctx, "nat", &ec2.NatGatewayArgs{
			AllocationId: eIp.ID(),
			SubnetId:     subOne.ID(),
		})

		if err != nil {
			return err
		}

		natTwo, err := ec2.NewNatGateway(ctx, "natTwo", &ec2.NatGatewayArgs{
			AllocationId: eIpTwo.ID(),
			SubnetId:     subTwo.ID(),
		})

		if err != nil {
			return err
		}

		privRt, err := ec2.NewRouteTable(ctx, "privateRouteTable", &ec2.RouteTableArgs{
			VpcId: vpc.ID(),
			Routes: ec2.RouteTableRouteArray{
				&ec2.RouteTableRouteArgs{
					CidrBlock:    pulumi.String("0.0.0.0/0"),
					NatGatewayId: nat.ID(), // us-east-1a
				},
			},
		})

		privRtTwo, err := ec2.NewRouteTable(ctx, "privateRouteTableTwo", &ec2.RouteTableArgs{
			VpcId: vpc.ID(),
			Routes: ec2.RouteTableRouteArray{
				&ec2.RouteTableRouteArgs{
					CidrBlock:    pulumi.String("0.0.0.0/0"),
					NatGatewayId: natTwo.ID(), // us-east-1b
				},
			},
		})

		// Associate the route table with the subnet
		_, err = ec2.NewRouteTableAssociation(ctx, "privateSubnetOneAssoc", &ec2.RouteTableAssociationArgs{
			SubnetId:     privateSubnetOne.ID(), // us-east-1a
			RouteTableId: privRt.ID(),           // nat us-east-1a
		})
		if err != nil {
			return err
		}

		_, err = ec2.NewRouteTableAssociation(ctx, "privateSubnetTwoAssoc", &ec2.RouteTableAssociationArgs{
			SubnetId:     privateSubnetTwo.ID(),
			RouteTableId: privRtTwo.ID(), // us-east-1b
		})
		if err != nil {
			return err
		}

		// load balancer with access to and from the internet.
		sg, err := ec2.NewSecurityGroup(ctx, "loadBalancerSg", &ec2.SecurityGroupArgs{
			Description: pulumi.String("Allow LoadBalancer inbound traffic"),
			VpcId:       vpc.ID(),
			Ingress: ec2.SecurityGroupIngressArray{
				&ec2.SecurityGroupIngressArgs{
					Description: pulumi.String("allow HTTP access from anywhere"),
					FromPort:    pulumi.Int(80),
					ToPort:      pulumi.Int(80),
					Protocol:    pulumi.String("tcp"),
					CidrBlocks: pulumi.StringArray{
						pulumi.String("0.0.0.0/0"),
					},
				},
				&ec2.SecurityGroupIngressArgs{
					Description: pulumi.String("allow HTTP access from anywhere"),
					FromPort:    pulumi.Int(80),
					ToPort:      pulumi.Int(80),
					Protocol:    pulumi.String("tcp"),
					Ipv6CidrBlocks: pulumi.StringArray{
						pulumi.String("::/0"),
					},
				},
			},
			Egress: ec2.SecurityGroupEgressArray{
				&ec2.SecurityGroupEgressArgs{
					Description: pulumi.String("allow connection to the outside"),
					FromPort:    pulumi.Int(0),
					ToPort:      pulumi.Int(443),
					Protocol:    pulumi.String("tcp"),
					CidrBlocks: pulumi.StringArray{
						pulumi.String("0.0.0.0/0"),
					},
				},
				&ec2.SecurityGroupEgressArgs{
					Description: pulumi.String("allow connection to the outside"),
					FromPort:    pulumi.Int(0),
					ToPort:      pulumi.Int(65535),
					Protocol:    pulumi.String("tcp"),
					Ipv6CidrBlocks: pulumi.StringArray{
						pulumi.String("::/0"),
					},
				},
			},
		})

		if err != nil {
			return err
		}

		// private sub SG
		// private access from lb and everything outbound.
		privateSg, err := ec2.NewSecurityGroup(ctx, "serviceSg", &ec2.SecurityGroupArgs{
			Description: pulumi.String("Allow LoadBalancer inbound traffic"),
			VpcId:       vpc.ID(),
			Ingress: ec2.SecurityGroupIngressArray{
				&ec2.SecurityGroupIngressArgs{
					Description:    pulumi.String("allow http from load balancer"),
					FromPort:       pulumi.Int(80),
					ToPort:         pulumi.Int(80),
					Protocol:       pulumi.String("tcp"),
					SecurityGroups: pulumi.StringArray{sg.ID()},
				},
			},
			Egress: ec2.SecurityGroupEgressArray{
				&ec2.SecurityGroupEgressArgs{
					Description: pulumi.String("allow connection to the outside v4"),
					FromPort:    pulumi.Int(80),
					ToPort:      pulumi.Int(443),
					Protocol:    pulumi.String("tcp"),
					CidrBlocks: pulumi.StringArray{
						pulumi.String("0.0.0.0/0"),
					},
				},
				&ec2.SecurityGroupEgressArgs{
					Description: pulumi.String("allow connection to the outside v6"),
					FromPort:    pulumi.Int(0),
					ToPort:      pulumi.Int(443),
					Protocol:    pulumi.String("tcp"),
					Ipv6CidrBlocks: pulumi.StringArray{
						pulumi.String("::/0"),
					},
				},
			},
		})

		if err != nil {
			return err
		}
		// END

		rtPublic, err := ec2.NewRouteTable(ctx, "publicRouteTable", &ec2.RouteTableArgs{
			VpcId: vpc.ID(),
			Routes: ec2.RouteTableRouteArray{
				&ec2.RouteTableRouteArgs{
					CidrBlock: pulumi.String("0.0.0.0/0"),
					GatewayId: ig.ID(),
				},
				&ec2.RouteTableRouteArgs{
					Ipv6CidrBlock: pulumi.String("::/0"),
					GatewayId:     ig.ID(),
				},
			},
		})

		rtPublicTwo, err := ec2.NewRouteTable(ctx, "publicRouteTableTwo", &ec2.RouteTableArgs{
			VpcId: vpc.ID(),
			Routes: ec2.RouteTableRouteArray{
				&ec2.RouteTableRouteArgs{
					CidrBlock: pulumi.String("0.0.0.0/0"),
					GatewayId: ig.ID(),
				},
				&ec2.RouteTableRouteArgs{
					Ipv6CidrBlock: pulumi.String("::/0"),
					GatewayId:     ig.ID(),
				},
			},
		})
		// Associate the route table with the subnet
		_, err = ec2.NewRouteTableAssociation(ctx, "tbAssociation", &ec2.RouteTableAssociationArgs{
			SubnetId:     subOne.ID(),
			RouteTableId: rtPublic.ID(),
		})
		if err != nil {
			return err
		}

		// Associate the route table with the subnet
		_, err = ec2.NewRouteTableAssociation(ctx, "publicTableAssociation", &ec2.RouteTableAssociationArgs{
			SubnetId:     subTwo.ID(),
			RouteTableId: rtPublicTwo.ID(),
		})
		if err != nil {
			return err
		}
		// Networking ends here //

		//Container
		containerPort := 80
		if param := cfg.GetInt("containerPort"); param != 0 {
			containerPort = param
		}
		cpu := 512
		if param := cfg.GetInt("cpu"); param != 0 {
			cpu = param
		}
		memory := 128
		if param := cfg.GetInt("memory"); param != 0 {
			memory = param
		}

		// An ECS cluster to deploy into
		cluster, err := ecs.NewCluster(ctx, "cluster", &ecs.ClusterArgs{
			Name: pulumi.String("control-tower"),
		})
		if err != nil {
			return err
		}

		args := lbx.ApplicationLoadBalancerArgs{
			Subnets: ec2.SubnetArray{
				subOne,
				subTwo,
			},
			SecurityGroups: pulumi.StringArray{
				sg.ID(),
			},
			DefaultTargetGroup: &lbx.TargetGroupArgs{
				HealthCheck: &lb.TargetGroupHealthCheckArgs{
					Path: pulumi.String("/health"),
				},
				Port:       pulumi.IntPtr(80),
				TargetType: pulumi.String("ip"),
			},
			Listener: &lbx.ListenerArgs{
				Port: pulumi.Int(80),
			},
		}
		// An ALB to serve the container endpoint to the internet
		loadbalancer, err := lbx.NewApplicationLoadBalancer(ctx, "loadbalancer", &args)
		if err != nil {
			return err
		}

		// An ECR repository to store our application's container image
		repo, err := ecrx.NewRepository(ctx, "ecrRepo", &ecrx.RepositoryArgs{
			ForceDelete: pulumi.Bool(true),
		})
		if err != nil {
			return err
		}

		// Build and publish our application's container image from ./app to the ECR repository
		image, err := ecrx.NewImage(ctx, "controTowerImage", &ecr.ImageArgs{
			RepositoryUrl: repo.Url,
			Context:       pulumi.String("./app"),
			Platform:      pulumi.String("linux/amd64"),
		})
		if err != nil {
			return err
		}

		// Deploy an ECS Service on Fargate to host the application container
		_, err = ecsx.NewFargateService(ctx, "service", &ecsx.FargateServiceArgs{
			Cluster: cluster.Arn,
			NetworkConfiguration: &ecs.ServiceNetworkConfigurationArgs{
				Subnets: pulumi.StringArray{
					privateSubnetOne.ID(),
					privateSubnetTwo.ID(),
				},
				SecurityGroups: pulumi.StringArray{
					privateSg.ID(),
				},
			},
			TaskDefinitionArgs: &ecsx.FargateServiceTaskDefinitionArgs{
				Container: &ecsx.TaskDefinitionContainerDefinitionArgs{
					Name:      pulumi.String("app"),
					Image:     image.ImageUri,
					Cpu:       pulumi.Int(cpu),
					Memory:    pulumi.Int(memory),
					Essential: pulumi.Bool(true),
					PortMappings: ecsx.TaskDefinitionPortMappingArray{
						&ecsx.TaskDefinitionPortMappingArgs{
							ContainerPort: pulumi.Int(containerPort),
							TargetGroup:   loadbalancer.DefaultTargetGroup,
						},
					},
					HealthCheck: &ecsx.TaskDefinitionHealthCheckArgs{
						Command: pulumi.StringArray{
							pulumi.String("CMD-SHELL"), pulumi.String("curl -f http://localhost:8080/ || exit 1"),
						},
						Interval: pulumi.Int(30),
						Retries:  pulumi.Int(3),
					},
				},
			},
		}, pulumi.DependsOn([]pulumi.Resource{repo, image}))
		if err != nil {
			return err
		}

		// The URL at which the container's HTTP endpoint will be available
		ctx.Export("url", pulumi.Sprintf("http://%s", loadbalancer.LoadBalancer.DnsName()))
		return nil
	})
}
