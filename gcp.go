package main

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/hashicorp/go-multierror"
	"google.golang.org/api/container/v1"
)

func GKEClusters(ctx context.Context, projectStr string) ([]Cluster, error) {
	if strings.TrimSpace(projectStr) == "" {
		return nil, nil
	}
	projects := strings.Split(projectStr, ",")
	containerSvc, err := container.NewService(ctx)
	if err != nil {
		return nil, err
	}
	clusterSvc := container.NewProjectsLocationsClustersService(containerSvc)
	clusters := make([]Cluster, 0)
	wg := multierror.Group{}
	lock := &sync.Mutex{}
	for _, p := range projects {
		project := p
		wg.Go(func() error {
			list := clusterSvc.List(fmt.Sprintf("projects/%s/locations/-", project))
			res, err := list.Do()
			if err != nil {
				return err
			}
			lock.Lock()
			defer lock.Unlock()
			for i := range res.Clusters {
				clusters = append(clusters, Cluster{
					Type:     "gke",
					Location: project,
					Region:   res.Clusters[i].Zone,
					Name:     res.Clusters[i].Name,
				})
			}
			return nil
		})
	}
	if err := wg.Wait().ErrorOrNil(); err != nil {
		return nil, err
	}
	return clusters, nil
}
