package gcloud

import (
	"context"
	"regexp"
	"strings"

	"github.com/google/go-containerregistry/pkg/authn"
	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	log "github.com/sirupsen/logrus"
)

func GetLatestTag(ctx context.Context, image string, regex string) (string, error) {
	repository, err := name.NewRepository(strings.Split(image, ":")[0])
	if err != nil {
		return "", err
	}
	tags, err := remote.List(repository, remote.WithAuthFromKeychain(authn.DefaultKeychain))
	if err != nil {
		return "", err
	}
	r, _ := regexp.Compile(regex)
	for _, tag := range tags {
		if r.MatchString(tag) {
			log.WithFields(log.Fields{
				"tag":  tag,
				"repo": strings.Split(image, ":")[0],
			}).Info("Found tag for update")
			return tag, nil
		}
	}
	log.WithFields(log.Fields{
		"repo": strings.Split(image, ":")[0],
	}).Info("Not found any tag for update")
	return "", nil
}
