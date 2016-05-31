package filter

import (
	"fmt"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/swarm/cluster"
	"github.com/docker/swarm/scheduler/node"
)

// AffinityFilter selects only nodes based on other containers on the node.
type AffinityFilter struct {
}

// Name returns the name of the filter
func (f *AffinityFilter) Name() string {
	return "affinity"
}

// Filter is exported
func (f *AffinityFilter) Filter(config *cluster.ContainerConfig, nodes []*node.Node, soft bool) ([]*node.Node, error) {
	affinities, err := parseExprs(config.Affinities())
	if err != nil {
		return nil, err
	}

	for _, affinity := range affinities {
		if !soft && affinity.isSoft {
			continue
		}
		log.Debugf("matching affinity: %s%s%s (soft=%t)", affinity.key, OPERATORS[affinity.operator], affinity.value, affinity.isSoft)

		candidates := []*node.Node{}
		for _, node := range nodes {
			switch affinity.key {
			case "container":
				containers := []string{}
				for _, container := range node.Containers {
					if len(container.Names) > 0 {
						containers = append(containers, container.Id, strings.TrimPrefix(container.Names[0], "/"))
					}
				}
				if affinity.Match(containers...) {
					candidates = append(candidates, node)
				}
			case "image":
				log.Infof("Start Case Affinity=Image %s", node.Addr)
				images := []string{}
				images_map := make(map[string]int)
				log.Infof("Start Range node.Images %s", node.Addr)
				log.Infof("Affinity Image %s", affinity.value)
				for _, image := range node.Images {
					//log.Infof("append image.ID %s", image.ID)
					images = append(images, image.ID)
					images_map[image.ID] = 1
					images = append(images, image.RepoTags...)
					for _, tag := range image.RepoTags {
						repo, _ := cluster.ParseRepositoryTag(tag)
						images = append(images, repo)
						images_map[repo] = 1
					}
				}
				log.Infof("End Range node.Images %s", node.Addr)
				log.Infof("Start Range node.PendingImages %s", node.Addr)
				for _, image := range node.PendingImages {
					images = append(images, image.ID)
					images_map[image.ID] = 1
					images = append(images, image.RepoTags...)
					for _, tag := range image.RepoTags {
						repo, _ := cluster.ParseRepositoryTag(tag)
						images = append(images, repo)
						images_map[repo] = 1
					}
				}
				log.Infof("End Range node.PendingImages %s", node.Addr)
				//log.Infof("Affinity Images: Total images on  %s = %v", node.Addr, len(node.Images))
				//log.Infof("Affinity Images: Total images on  %s = %v", node.Addr, len(node.PendingImages))
				log.Infof("Start Match(map) %s", affinity.value)
				if _, ok := images_map[affinity.value]; ok {
					log.Infof("Match Image %s", node.Addr)
					candidates = append(candidates, node)
				}
				log.Infof("End Match(map) %s", affinity.value)
				//log.Infof("Start Match(images...) %s", node.Addr)
				//if affinity.Match(images...) {
				//	candidates = append(candidates, node)
				//}
				//log.Infof("End Match(images...) %s", node.Addr)
				log.Infof("End Case Affinity=Image %s", node.Addr)
			default:
				labels := []string{}
				for _, container := range node.Containers {
					labels = append(labels, container.Labels[affinity.key])
				}
				if affinity.Match(labels...) {
					candidates = append(candidates, node)
				}

			}
		}
		if len(candidates) == 0 {
			return nil, fmt.Errorf("unable to find a node that satisfies the affinity %s%s%s", affinity.key, OPERATORS[affinity.operator], affinity.value)
		}
		nodes = candidates
	}

	return nodes, nil
}

// GetFilters returns a list of the affinities found in the container config.
func (f *AffinityFilter) GetFilters(config *cluster.ContainerConfig) ([]string, error) {
	allAffinities := []string{}
	affinities, err := parseExprs(config.Affinities())
	if err != nil {
		return nil, err
	}
	for _, affinity := range affinities {
		allAffinities = append(allAffinities, fmt.Sprintf("%s%s%s (soft=%t)", affinity.key, OPERATORS[affinity.operator], affinity.value, affinity.isSoft))
	}
	return allAffinities, nil
}
