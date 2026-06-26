package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	"github.com/cert-manager/cert-manager/pkg/acme/webhook/cmd"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	"github.com/cert-manager/cert-manager/pkg/issuer/acme/dns/util"
	"github.com/blackdark/cert-manager-webhook-dynu/dynu"
)

// GroupName is the API group name (should be unique cluster-wide).
var GroupName = os.Getenv("GROUP_NAME")

type actionType int

const (
	actionPresent actionType = iota
	actionCleanup
)

var actionNames = map[actionType]string{
	actionPresent: "Present",
	actionCleanup: "Cleanup",
}

func main() {
	if GroupName == "" {
		panic("GROUP_NAME must be specified")
	}

	flag.Set("v", "2")

	cmd.RunWebhookServer(GroupName,
		&dynuDNSProviderSolver{},
	)
}

type dynuDNSProviderSolver struct {
	client *kubernetes.Clientset
}

type dynuDNSProviderConfig struct {
	APIKeySecretRef cmmeta.SecretKeySelector `json:"apiKeySecretRef"`
}

func (c *dynuDNSProviderSolver) Name() string {
	return "dynu"
}

func (c *dynuDNSProviderSolver) Present(ch *v1alpha1.ChallengeRequest) error {
	err := c.doAction(ch, actionPresent)
	if err != nil {
		klog.Errorf("Error while presenting record `%s`: %v", ch.ResolvedFQDN, err)
		klog.Flush()
		return err
	}
	klog.Flush()
	return nil
}

func (c *dynuDNSProviderSolver) CleanUp(ch *v1alpha1.ChallengeRequest) error {
	err := c.doAction(ch, actionCleanup)
	if err != nil {
		klog.Errorf("Error while cleaning up record `%s`: %v", ch.ResolvedFQDN, err)
		klog.Flush()
		return err
	}
	klog.Flush()
	return nil
}

func (c *dynuDNSProviderSolver) Initialize(kubeClientConfig *rest.Config, stopCh <-chan struct{}) error {
	cl, err := kubernetes.NewForConfig(kubeClientConfig)
	if err != nil {
		return err
	}
	c.client = cl
	return nil
}

func (c *dynuDNSProviderSolver) getSecretKey(secret cmmeta.SecretKeySelector, namespace string) (string, error) {
	klog.V(6).Infof("retrieving key `%s` in secret `%s/%s`", secret.Key, namespace, secret.Name)

	sec, err := c.client.CoreV1().Secrets(namespace).Get(context.Background(), secret.Name, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("secret %q/%q: %w", namespace, secret.Name, err)
	}

	data, ok := sec.Data[secret.Key]
	if !ok {
		return "", fmt.Errorf("key %q not found in secret %s/%s", secret.Key, namespace, secret.Name)
	}

	return string(data), nil
}

func (c *dynuDNSProviderSolver) doAction(ch *v1alpha1.ChallengeRequest, action actionType) error {
	cfg, err := loadConfig(ch.Config)
	if err != nil {
		return err
	}
	klog.V(2).Infof("%s TXT record `%s`", actionNames[action], ch.ResolvedFQDN)

	fqdn := util.UnFqdn(ch.ResolvedFQDN)
	key := ch.Key

	apiKey, err := c.getSecretKey(cfg.APIKeySecretRef, ch.ResourceNamespace)
	if err != nil {
		return err
	}

	api := &dynu.API{APIKey: apiKey}
	ctx := context.Background()

	domainID, nodeName, err := api.ResolveZone(ctx, fqdn)
	if err != nil {
		return err
	}

	switch action {
	case actionPresent:
		added, err := api.AddTXTRecord(ctx, domainID, nodeName, key)
		if err != nil {
			return err
		}
		if added {
			klog.V(2).Infof("Successfully presented TXT record `%s`", ch.ResolvedFQDN)
		} else {
			klog.V(2).Infof("TXT record `%s` already exists, skipping", ch.ResolvedFQDN)
		}
	case actionCleanup:
		deleted, err := api.DeleteTXTRecord(ctx, domainID, key)
		if err != nil {
			return err
		}
		if deleted {
			klog.V(2).Infof("Successfully cleaned up TXT record `%s`", ch.ResolvedFQDN)
		} else {
			klog.V(2).Infof("TXT record `%s` not found, skipping cleanup", ch.ResolvedFQDN)
		}
	}
	return nil
}

func loadConfig(cfgJSON *extapi.JSON) (dynuDNSProviderConfig, error) {
	cfg := dynuDNSProviderConfig{}
	if cfgJSON == nil {
		return cfg, fmt.Errorf("solver config is required and must include apiKeySecretRef")
	}
	if err := json.Unmarshal(cfgJSON.Raw, &cfg); err != nil {
		return cfg, fmt.Errorf("error decoding solver config: %v", err)
	}
	if cfg.APIKeySecretRef.Name == "" || cfg.APIKeySecretRef.Key == "" {
		return cfg, fmt.Errorf("invalid solver config: apiKeySecretRef.name and apiKeySecretRef.key are required")
	}
	return cfg, nil
}
