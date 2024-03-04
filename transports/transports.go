/*
 * Copyright (c) 2014, Yawning Angel <yawning at torproject dot org>
 * All rights reserved.
 *
 * Redistribution and use in source and binary forms, with or without
 * modification, are permitted provided that the following conditions are met:
 *
 *  * Redistributions of source code must retain the above copyright notice,
 *    this list of conditions and the following disclaimer.
 *
 *  * Redistributions in binary form must reproduce the above copyright notice,
 *    this list of conditions and the following disclaimer in the documentation
 *    and/or other materials provided with the distribution.
 *
 * THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
 * AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
 * IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
 * ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE
 * LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
 * CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
 * SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
 * INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
 * CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
 * ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
 * POSSIBILITY OF SUCH DAMAGE.
 */

// Package transports provides a interface to query supported pluggable
// transports.
package transports

import (
	"encoding/json"
	"errors"
	"os"

	Optimizer "github.com/OperatorFoundation/Optimizer-go/Optimizer/v3"
	replicant "github.com/OperatorFoundation/Replicant-go/Replicant/v3"
	"github.com/OperatorFoundation/Replicant-go/Replicant/v3/polish"
	"github.com/OperatorFoundation/Replicant-go/Replicant/v3/toneburst"
	"github.com/OperatorFoundation/Shadow-go/shadow/v3"
	"github.com/OperatorFoundation/Starbridge-go/Starbridge/v3"
	"golang.org/x/net/proxy"
)

// Transports returns the list of registered transport protocols.
func Transports() []string {
	return []string{"shadow", "Replicant", "Starbridge", "Optimizer"}
}

func ParseArgsShadow(args string) (*shadow.Transport, error) {
	var config shadow.ClientConfig
	bytes := []byte(args)
	jsonError := json.Unmarshal(bytes, &config)
	if jsonError != nil {
		return nil, errors.New("shadow options json decoding error")
	}
	transport := shadow.NewTransport(config.Password, config.CipherName, config.Address)

	return &transport, nil
}

func ParseArgsShadowServer(args string) (*shadow.ServerConfig, error) {
	var config shadow.ServerConfig

	bytes := []byte(args)
	jsonError := json.Unmarshal(bytes, &config)
	if jsonError != nil {
		return nil, errors.New("shadow server options json decoding error")
	}

	return &config, nil
}

func CreateDefaultReplicantServer() replicant.ServerConfig {
	config := replicant.ServerConfig{
		Toneburst: nil,
		Polish:    nil,
	}

	return config
}

func CreateReplicantConfigs(address string, isToneburst bool, isPolish bool, bindAddress *string) error {
	var err error
	serverConfig := CreateDefaultReplicantServer()
	toneburstConfig := toneburst.WhalesongConfig{
		AddSequences:    []toneburst.Sequence{},
		RemoveSequences: []toneburst.Sequence{},
	}

	polishServerConfig, err := polish.NewSilverServerConfig()
	if err != nil {
		return err
	}

	serverConfig.Toneburst = toneburstConfig
	serverConfig.Polish = polishServerConfig
	encodedServerConf, err := serverConfig.Encode()
	if err != nil {
		return err
	}

	var sConfig replicant.ServerJSONOuterConfig
	sConfig.Replicant.Config = encodedServerConf
	serverJsonBytes, err := json.MarshalIndent(sConfig, "", "\t")
	if err != nil {
		return err
	}

	polishClientConfig, err := polish.NewSilverClientConfig(polishServerConfig)
	if err != nil {
		return err
	}

	config := replicant.ClientConfig{
		Toneburst: toneburstConfig,
		Polish:    polishClientConfig,
		Address:   address,
	}

	clientConfigs, err := config.Encode()
	if err != nil {
		return err
	}

	var cConfig struct {
		Config string `json:"config"`
	}
	cConfig.Config = clientConfigs
	clientJsonBytes, err := json.MarshalIndent(cConfig, "", "\t")
	if err != nil {
		return err
	}

	err = os.WriteFile("ReplicantServerConfigV3.json", serverJsonBytes, 0777)
	if err != nil {
		return err
	}

	err = os.WriteFile("ReplicantClientConfigV3.json", clientJsonBytes, 0777)
	if err != nil {
		return err
	}
	return nil
}

func ParseArgsReplicantClient(args string, dialer proxy.Dialer) (*replicant.TransportClient, error) {
	var config *replicant.ClientConfig

	var ReplicantConfig replicant.ClientJSONConfig
	if args == "" {
		return nil, errors.New("must specify transport options when using replicant")
	}
	argsBytes := []byte(args)
	unmarshalError := json.Unmarshal(argsBytes, &ReplicantConfig)
	if unmarshalError != nil {
		return nil, errors.New("could not unmarshal Replicant args")
	}
	var parseErr error
	config, parseErr = replicant.DecodeClientConfig(ReplicantConfig.Config)
	if parseErr != nil {
		return nil, errors.New("could not parse config")
	}

	transport := replicant.TransportClient{
		Config:  *config,
		Address: (*config).Address,
		Dialer:  dialer,
	}

	return &transport, nil
}

// target string, dialer proxy.Dialer
func ParseArgsReplicantServer(args string) (*replicant.ServerConfig, error) {
	var config *replicant.ServerConfig

	type replicantJsonConfig struct {
		Config string
	}
	var ReplicantConfig replicantJsonConfig
	if args == "" {
		transport := CreateDefaultReplicantServer()
		return &transport, nil
	}
	argsBytes := []byte(args)
	unmarshalError := json.Unmarshal(argsBytes, &ReplicantConfig)
	if unmarshalError != nil {
		return nil, errors.New("could not unmarshal Replicant args")
	}
	var parseErr error
	config, parseErr = replicant.DecodeServerConfig(ReplicantConfig.Config)
	if parseErr != nil {
		return nil, parseErr
	}

	return config, nil
}

func ParseArgsStarbridgeClient(args string, dialer proxy.Dialer) (*Starbridge.TransportClient, error) {
	var config Starbridge.ClientConfig
	bytes := []byte(args)
	jsonError := json.Unmarshal(bytes, &config)
	if jsonError != nil {
		return nil, errors.New("starbridge options json decoding error")
	}
	transport := Starbridge.TransportClient{
		Config:  config,
		Address: config.Address,
		Dialer:  dialer,
	}

	return &transport, nil
}

func ParseArgsStarbridgeServer(args string) (*Starbridge.ServerConfig, error) {
	var config Starbridge.ServerConfig

	bytes := []byte(args)
	jsonError := json.Unmarshal(bytes, &config)
	if jsonError != nil {
		return nil, errors.New("starbridge server options json decoding error")
	}

	return &config, nil
}

type OptimizerConfig struct {
	Transports []interface{} `json:"transports"`
	Strategy   string        `json:"strategy"`
}

type OptimizerArgs struct {
	Address string                 `json:"address"`
	Name    string                 `json:"name"`
	Config  map[string]interface{} `json:"config"`
}

func ParseArgsOptimizer(jsonConfig string, dialer proxy.Dialer) (*Optimizer.Client, error) {
	var config OptimizerConfig
	var transports []Optimizer.TransportDialer
	var strategy Optimizer.Strategy
	jsonByte := []byte(jsonConfig)
	parseErr := json.Unmarshal(jsonByte, &config)
	if parseErr != nil {
		return nil, errors.New("could not marshal optimizer config")
	}
	transports, parseErr = parseTransports(config.Transports, dialer)
	if parseErr != nil {
		println("this is the returned error from parseTransports:", parseErr)
		return nil, errors.New("could not parse transports")
	}

	strategy, parseErr = parseStrategy(config.Strategy, transports)
	if parseErr != nil {
		return nil, errors.New("could not parse strategy")
	}

	transport := Optimizer.NewOptimizerClient(transports, strategy)

	return transport, nil
}

func parseStrategy(strategyString string, transports []Optimizer.TransportDialer) (Optimizer.Strategy, error) {
	switch strategyString {
	case "first":
		strategy := Optimizer.NewFirstStrategy(transports)
		return strategy, nil
	case "random":
		strategy := Optimizer.NewRandomStrategy(transports)
		return strategy, nil
	case "rotate":
		strategy := Optimizer.NewRotateStrategy(transports)
		return strategy, nil
	case "track":
		return Optimizer.NewTrackStrategy(transports), nil
	case "minimizeDialDuration":
		return Optimizer.NewMinimizeDialDuration(transports), nil

	default:
		return nil, errors.New("invalid strategy")
	}
}

func parseTransports(otcs []interface{}, dialer proxy.Dialer) ([]Optimizer.TransportDialer, error) {
	transports := make([]Optimizer.TransportDialer, len(otcs))
	for index, untypedOtc := range otcs {
		switch untypedOtc.(type) {
		case map[string]interface{}:
			otc := untypedOtc.(map[string]interface{})
			transport, err := parsedTransport(otc, dialer)
			if err != nil {
				return nil, errors.New("transport could not parse config")
				//this error sucks and is uninformative
			}
			transports[index] = transport
		default:
			return nil, errors.New("unsupported type for transport")
		}

	}
	return transports, nil
}

func parsedTransport(otc map[string]interface{}, dialer proxy.Dialer) (Optimizer.TransportDialer, error) {
	var config map[string]interface{}

	type PartialOptimizerConfig struct {
		Name string `json:"name"`
	}
	jsonString, MarshalErr := json.Marshal(otc)
	if MarshalErr != nil {
		return nil, errors.New("error marshalling optimizer otc")
	}
	var PartialConfig PartialOptimizerConfig
	unmarshalError := json.Unmarshal(jsonString, &PartialConfig)
	if unmarshalError != nil {
		return nil, errors.New("error unmarshalling optimizer otc")
	}
	//on to parsing the config
	untypedConfig, ok3 := otc["config"]
	if !ok3 {
		return nil, errors.New("missing config in transport parser")
	}

	switch untypedConfig.(type) {

	case map[string]interface{}:
		config = untypedConfig.(map[string]interface{})

	default:
		return nil, errors.New("unsupported type for optimizer config option")
	}

	jsonConfigBytes, configMarshalError := json.Marshal(config)
	if configMarshalError != nil {
		return nil, errors.New("could not marshal Optimizer config")
	}
	jsonConfigString := string(jsonConfigBytes)
	switch PartialConfig.Name {
	case "shadow":
		shadowTransport, parseErr := ParseArgsShadow(jsonConfigString)
		if parseErr != nil {
			return nil, errors.New("could not parse shadow Args")
		}
		return shadowTransport, nil
	case "Replicant":
		replicantTransport, parseErr := ParseArgsReplicantClient(jsonConfigString, dialer)
		if parseErr != nil {
			return nil, errors.New("could not parse replicant Args")
		}
		return replicantTransport, nil
	case "Starbridge":
		starbridgeTransport, parseErr := ParseArgsStarbridgeClient(jsonConfigString, dialer)
		if parseErr != nil {
			return nil, errors.New("could not parse starbridge Args")
		}
		return starbridgeTransport, nil
	case "Optimizer":
		optimizerTransport, parseErr := ParseArgsOptimizer(jsonConfigString, dialer)
		if parseErr != nil {
			return nil, errors.New("could not parse Optimizer Args")
		}
		return optimizerTransport, nil
	default:
		return nil, errors.New("unsupported transport name")
	}
}
