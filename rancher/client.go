package rancher

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// WaitInterval in seconds between upgrade attempts
	WaitInterval int = 2
)

// ConfigItemCollection model
type ConfigItemCollection struct {
	List []ConfigItem `json:"data"`
}

// ConfigItem model
type ConfigItem struct {
	ID                     string         `json:"id"`
	Name                   string         `json:"name"`
	State                  string         `json:"state"`
	HealthState            string         `json:"healthState"`
	LaunchConfig           LaunchConfig   `json:"launchConfig"`
	SecondaryLaunchConfigs []LaunchConfig `json:"secondaryLaunchConfigs"`
}

// LaunchConfig model
type LaunchConfig map[string]interface{}

// Environment model
type Environment struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Stack model
type Stack struct {
	ID    string `json:"id"`
	EnvID string `json:"envId"`
	Name  string `json:"name"`
}

// Service model
type Service struct {
	ID                     string         `json:"id"`
	EnvID                  string         `json:"envId"`
	StackID                string         `json:"stackId"`
	Name                   string         `json:"name"`
	State                  string         `json:"state"`
	HealthState            string         `json:"healthState"`
	LaunchConfig           LaunchConfig   `json:"launchConfig"`
	SecondaryLaunchConfigs []LaunchConfig `json:"secondaryLaunchConfigs"`
}

// UpgradeRequest model
type UpgradeRequest struct {
	InServiceStrategy InServiceStrategy `json:"inServiceStrategy,omitempty"`
}

// InServiceStrategy model
type InServiceStrategy struct {
	BatchSize              int            `json:"batchSize,omitempty"`
	IntervalMills          int            `json:"intervalMillis,omitempty"`
	StartFirst             bool           `json:"startFirst,omitempty"`
	LaunchConfig           LaunchConfig   `json:"launchConfig,omitempty"`
	SecondaryLaunchConfigs []LaunchConfig `json:"secondaryLaunchConfigs,omitempty"`
}

// Client for rancher REST api
type Client struct {
	debug bool
	url   string
}

// NewClient constructor
func NewClient(debug bool, baseURL string, key string, secret string) (*Client, error) {
	baseURL = strings.TrimSpace(baseURL)
	apiBaseURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}

	key = strings.TrimSpace(key)
	if key == "" {
		return nil, fmt.Errorf("argument [key] can't be blank")
	}

	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil, fmt.Errorf("argument [secret] can't be blank")
	}

	api := &Client{}
	api.debug = debug

	if apiBaseURL.Scheme == "https" {
		http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	api.url = fmt.Sprintf("%s://%s:%s@%s/v1", apiBaseURL.Scheme, key, secret, apiBaseURL.Host)
	if debug {
		log.Printf("Rancher api url [%s]\n", api.url)
	}

	return api, nil
}

func (api *Client) fetchItems(uri string) (*[]ConfigItem, error) {
	targetURL := fmt.Sprintf("%s/%s", api.url, uri)
	if api.debug {
		log.Printf("Fetch list items from [%s]\n", targetURL)
	}
	resp, err := http.Get(targetURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	items := &ConfigItemCollection{}
	err = json.NewDecoder(resp.Body).Decode(items)
	if err != nil {
		return nil, err
	}

	return &items.List, nil
}

// GetEnv info from rancher
func (api *Client) GetEnv(envName string) (*Environment, error) {
	envName = strings.TrimSpace(envName)
	if envName == "" {
		return nil, fmt.Errorf("argument [envName] can't be blank")
	}

	items, err := api.fetchItems("projects?limit=1000")
	if err != nil {
		return nil, err
	}

	var env *Environment
	for _, el := range *items {
		if envName == el.Name {
			env = &Environment{
				el.ID,
				el.Name,
			}
			break
		}
	}

	if env == nil {
		return nil, fmt.Errorf("environment [%s] doesn't exist in Rancher, or your API credentials don't have access to it", envName)
	}

	return env, nil
}

// GetStack info from rancher
func (api *Client) GetStack(envID string, stackName string) (*Stack, error) {
	envID = strings.TrimSpace(envID)
	if envID == "" {
		return nil, fmt.Errorf("argument [envID] can't be blank")
	}

	stackName = strings.TrimSpace(stackName)
	if stackName == "" {
		return nil, fmt.Errorf("argument [stackName] can't be blank")
	}

	items, err := api.fetchItems(fmt.Sprintf("projects/%s/environments?limit=1000", envID))
	if err != nil {
		return nil, err
	}

	var stack *Stack
	for _, el := range *items {
		if stackName == el.Name {
			stack = &Stack{
				el.ID,
				envID,
				el.Name,
			}
			break
		}
	}

	if stack == nil {
		return nil, fmt.Errorf("stack [%s] doesn't exist in Rancher, or your API credentials don't have access to it", stackName)
	}

	return stack, nil
}

// GetService info from rancher
func (api *Client) GetService(envID string, stackID string, serviceName string) (*Service, error) {
	envID = strings.TrimSpace(envID)
	if envID == "" {
		return nil, fmt.Errorf("argument [envID] can't be blank")
	}

	stackID = strings.TrimSpace(stackID)
	if stackID == "" {
		return nil, fmt.Errorf("argument [stackID] can't be blank")
	}

	serviceName = strings.TrimSpace(serviceName)
	if serviceName == "" {
		return nil, fmt.Errorf("argument [serviceName] can't be blank")
	}

	items, err := api.fetchItems(fmt.Sprintf("projects/%s/environments/%s/services?limit=1000", envID, stackID))
	if err != nil {
		return nil, err
	}

	var srv *Service
	for _, el := range *items {
		if serviceName == el.Name {
			srv = &Service{
				el.ID,
				envID,
				stackID,
				el.Name,
				el.State,
				el.HealthState,
				el.LaunchConfig,
				el.SecondaryLaunchConfigs,
			}
			break
		}
	}

	if srv == nil {
		return nil, fmt.Errorf("service [%s] doesn't exist in Rancher, or your API credentials don't have access to it", serviceName)
	}

	return srv, nil
}

// GetServiceStatus info from rancher
func (api *Client) GetServiceStatus(service *Service) (string, error) {
	if service == nil {
		return "", fmt.Errorf("argument [service] can't be null")
	}

	resp, err := http.Get(fmt.Sprintf("%s/projects/%s/services/%s", api.url, service.EnvID, service.ID))
	if err != nil {
		if api.debug {
			log.Println(err)
		}
		return "", fmt.Errorf("Unable to request the service status from the Rancher API")
	}
	defer resp.Body.Close()

	srv := &Service{}
	err = json.NewDecoder(resp.Body).Decode(srv)
	if err != nil {
		return "", err
	}

	if srv.HealthState == "unhealthy" {
		return "unhealthy", nil
	}

	return srv.State, nil
}

// FinishUpgrade rancher service
func (api *Client) FinishUpgrade(service *Service) error {
	if service == nil {
		return fmt.Errorf("argument [service] can't be null")
	}

	resp, err := http.Post(fmt.Sprintf("%s/projects/%s/services/%s/?action=finishupgrade", api.url, service.EnvID, service.ID), "application/json; charset=utf-8", bytes.NewBuffer([]byte(`{}`)))
	if err != nil {
		if api.debug {
			log.Println(err)
		}
		return fmt.Errorf("Unable to finish the previous upgrade in Rancher API")
	}
	defer resp.Body.Close()

	return nil
}

// RollbackUpgrade rancher service
func (api *Client) RollbackUpgrade(service *Service) error {
	if service == nil {
		return fmt.Errorf("argument [service] can't be null")
	}

	resp, err := http.Post(fmt.Sprintf("%s/projects/%s/services/%s/?action=rollback", api.url, service.EnvID, service.ID), "application/json; charset=utf-8", bytes.NewBuffer([]byte(`{}`)))
	if err != nil {
		if api.debug {
			log.Println(err)
		}
		return fmt.Errorf("Unable to finish rollback in Rancher API")
	}
	defer resp.Body.Close()

	return nil
}

// WaitForServiceState rancher service
func (api *Client) WaitForServiceState(service *Service, timeout int, targetState string) error {
	var state string
	var err error

	state = service.State
	for attempts := 0; attempts < timeout; attempts += WaitInterval {
		time.Sleep(time.Duration(WaitInterval) * time.Second)

		state, err = api.GetServiceStatus(service)
		if err != nil {
			log.Println(err)
		}
		if api.debug {
			log.Printf("Current service state = [%s]\n", state)
		}
		if state == targetState {
			return nil
		}
	}

	return fmt.Errorf("Upgrade error: current service state [%s], but it needs to be '%s'", state, targetState)
}

// NewUpgradeRequest constructor
func (api *Client) NewUpgradeRequest(
	service *Service,
	batchSize int,
	batchInterval int,
	startBeforeStop bool,
	image string,
	sidekicks bool,
	newSidekickImage *SidekickImageParams) (*UpgradeRequest, error) {

	image = strings.TrimSpace(image)
	if image == "" {
		return nil, fmt.Errorf("argument [image] can't be blank")
	}

	if service == nil {
		return nil, fmt.Errorf("argument [service] can't be null")
	}

	req := &UpgradeRequest{}
	req.InServiceStrategy = InServiceStrategy{
		BatchSize:     batchSize,
		IntervalMills: batchInterval * 1000,
		StartFirst:    startBeforeStop,
		LaunchConfig:  service.LaunchConfig,
	}
	req.InServiceStrategy.LaunchConfig["imageUuid"] = fmt.Sprintf("docker:%s", image)
	req.InServiceStrategy.LaunchConfig["secrets"] = new([0]string)

	// Add current Sidekick configuration to upgrade request if it present
	if sidekicks {
		req.InServiceStrategy.SecondaryLaunchConfigs = service.SecondaryLaunchConfigs
	}

	// Return current upgrade request state if NOT need upgrade Sidekick image
	if len(*newSidekickImage) == 0 {
		return req, nil
	}

	// Add new Sidekick configuration to upgrade request
	for i, cfg := range req.InServiceStrategy.SecondaryLaunchConfigs {
		for _, img := range *newSidekickImage {
			if cfg["name"] == img.Name {
				req.InServiceStrategy.SecondaryLaunchConfigs[i]["imageUuid"] = fmt.Sprintf("docker:%s", img.Name)
				break
			}
		}
	}

	return req, nil
}

// UpgradeService rancher service
func (api *Client) UpgradeService(service *Service, req *UpgradeRequest) error {
	if service == nil {
		return fmt.Errorf("argument [service] can't be null")
	}

	if req == nil {
		return fmt.Errorf("argument [req] can't be null")
	}

	// Marshal request body
	b, err := json.Marshal(req)
	if err != nil {
		return err
	}
	buff := bytes.NewBuffer(b)

	resp, err := http.Post(fmt.Sprintf("%s/projects/%s/services/%s/?action=upgrade", api.url, service.EnvID, service.ID), "application/json; charset=utf-8", buff)
	if err != nil {
		if api.debug {
			log.Println(err)
			log.Printf("Service upgrade HttpStatus code = [%d]\n", resp.StatusCode)
		}
		return fmt.Errorf("Unable to upgrade service from the Rancher API")
	}
	defer resp.Body.Close()

	if api.debug {
		log.Printf("Service upgrade HttpStatus code = [%d]\n", resp.StatusCode)
	}

	return nil
}
