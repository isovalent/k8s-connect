package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/c-bata/go-prompt"
	"github.com/hashicorp/go-multierror"
)

type Cluster struct {
	Type     string
	Location string
	Region   string
	Name     string
}

var (
	gcpProjects string
	suggests    []prompt.Suggest
	clusterMap  map[string]Cluster
	exit        bool
)

func main() {
	ctx := context.Background()
	flag.StringVar(&gcpProjects, "projects", "", "GCP comma separated projects list")
	flag.Parse()

	log.Println("reading clusters ...")
	wg := multierror.Group{}
	var (
		gkeClusters, eksClusters []Cluster
	)
	wg.Go(func() error {
		clusters, err := GKEClusters(ctx, gcpProjects)
		if err != nil {
			log.Printf("unable to read eks clusters: %s\n", err)
			return err
		}
		gkeClusters = clusters
		return nil
	})
	wg.Go(func() error {
		clusters, err := EKSClusters(ctx)
		if err != nil {
			log.Printf("unable to read eks clusters: %s\n", err)
			return err
		}
		eksClusters = clusters
		return nil
	})
	if err := wg.Wait().ErrorOrNil(); err != nil {
		log.Fatalf("failed: %s", err)
	}
	clusters := append(gkeClusters, eksClusters...)
	sort.Slice(clusters, func(i, j int) bool {
		return (strings.Compare(clusters[i].Type, clusters[j].Type) == 0) &&
			(strings.Compare(clusters[i].Location, clusters[j].Location) == 0) &&
			(strings.Compare(clusters[i].Region, clusters[j].Region) == 0) &&
			(strings.Compare(clusters[i].Name, clusters[j].Name) == 0)
	})
	clusterMap = make(map[string]Cluster, len(clusters))
	for i, c := range clusters {
		key := fmt.Sprintf("%d. %s", i+1, c.Name)
		clusterMap[key] = c
		suggests = append(suggests, prompt.Suggest{Text: key, Description: fmt.Sprintf("%s : %s : %s", c.Type, c.Location, c.Region)})
	}
	p := prompt.New(
		executor,
		completer,
		prompt.OptionPrefix("(.exit)> "),
		prompt.OptionTitle("K8S connector"),
		prompt.OptionCompletionOnDown(),
		prompt.OptionShowCompletionAtStart(),
		prompt.OptionSetExitCheckerOnInput(exitChecker),
	)
	p.Run()
}

func completer(in prompt.Document) []prompt.Suggest {
	return prompt.FilterFuzzy(suggests, in.GetWordBeforeCursor(), true)
}

func executor(in string) {
	if in == "" {
		return
	}
	cluster, ok := clusterMap[in]
	if !ok {
		log.Fatalf("cluster not found: %s", in)
	}
	var cmd *exec.Cmd
	switch cluster.Type {
	case "gke":
		log.Printf("gcloud container clusters get-credentials %s --project %s --region %s\n", cluster.Name, cluster.Location, cluster.Region)
		cmd = exec.Command("gcloud", "container", "clusters", "get-credentials", cluster.Name, "--project", cluster.Location, "--region", cluster.Region)
	case "eks":
		log.Printf("aws eks --profile %s --region %s update-kubeconfig --name %s\n", cluster.Location, cluster.Region, cluster.Name)
		cmd = exec.Command("aws", "eks", "--profile", cluster.Location, "--region", cluster.Region, "update-kubeconfig", "--name", cluster.Name)
	}
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stdout
	if err := cmd.Run(); err != nil {
		log.Fatalf("unable connect to cluster: %s", err)
	}
	exit = true
}

func exitChecker(in string, _ bool) bool {
	return exit || in == ".exit"
}
