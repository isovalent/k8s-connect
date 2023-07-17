package main

import (
	"context"
	"os"
	"regexp"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/eks"
	"github.com/hashicorp/go-multierror"
)

func EKSClusters(ctx context.Context) ([]Cluster, error) {
	profiles, err := readProfiles()
	if err != nil {
		return nil, err
	}
	clients, err := createClients(ctx, profiles)
	if err != nil {
		return nil, err
	}
	result := make([]Cluster, 0)
	wg := multierror.Group{}
	lock := &sync.Mutex{}
	for profile := range clients {
		for _, region := range regions {
			profile, region := profile, region
			wg.Go(func() error {
				clusters, err := readClusters(ctx, clients[profile], profile, region)
				if err != nil {
					return err
				}
				lock.Lock()
				defer lock.Unlock()
				result = append(result, clusters...)
				return nil
			})
		}
	}
	if err := wg.Wait().ErrorOrNil(); err != nil {
		return nil, err
	}
	return result, nil
}

// TODO: avoid hardcoded regions
var (
	profilePrefix       = "[profile "
	profileSuffix       = "]"
	defaultProfileRegex = regexp.MustCompile(`^\[default]$`)
	profileRegex        = regexp.MustCompile(`^\` + profilePrefix + `.*` + profileSuffix + `$`)
	regions             = []string{
		"us-east-1", "us-east-2", "us-west-1", "us-west-2", "ap-south-1", "ap-northeast-3", "ap-northeast-2",
		"ap-southeast-1", "ap-southeast-2", "ap-northeast-1", "ca-central-1", "eu-central-1", "eu-west-1",
		"eu-west-2", "eu-west-3", "eu-north-1", "sa-east-1",
	}
)

func readProfiles() ([]string, error) {
	cfg, err := os.ReadFile(config.DefaultSharedConfigFilename())
	if err != nil {
		return nil, err
	}
	profiles := make([]string, 0)
	for _, l := range strings.Split(string(cfg), "\n") {
		if defaultProfileRegex.MatchString(l) {
			profiles = append(profiles, "default")
			continue
		}
		if profileRegex.MatchString(l) {
			profile := strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(l), profilePrefix), profileSuffix)
			profiles = append(profiles, strings.TrimSpace(profile))
		}
	}
	return profiles, nil
}

func createClients(ctx context.Context, profiles []string) (map[string]*eks.Client, error) {
	clients := make(map[string]*eks.Client, len(profiles))
	for _, p := range profiles {
		cfg, err := config.LoadDefaultConfig(ctx, clientOpts(p)...)
		if err != nil {
			return nil, err
		}
		clients[p] = eks.NewFromConfig(cfg)
	}
	return clients, nil
}

func clientOpts(account string) []func(*config.LoadOptions) error {
	var opts []func(*config.LoadOptions) error
	if account != "default" {
		opts = append(opts, config.WithSharedConfigProfile(account))
	}
	return opts
}

func readClusters(ctx context.Context, client *eks.Client, profile, region string) ([]Cluster, error) {
	res, err := client.ListClusters(ctx, &eks.ListClustersInput{}, func(options *eks.Options) {
		options.Region = region
	})
	if err != nil {
		return nil, err
	}
	clusters := make([]Cluster, 0)
	for _, c := range res.Clusters {
		clusters = append(clusters, Cluster{
			Type:     "eks",
			Location: profile,
			Region:   region,
			Name:     c,
		})
	}
	return clusters, nil
}
