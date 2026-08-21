package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/datacenter/internal/simulator"
	modbusAdapter "github.com/datacenter/internal/simulator/modbus"
	mqttAdapter "github.com/datacenter/internal/simulator/mqtt"
	opcuaAdapter "github.com/datacenter/internal/simulator/opcua"
	tcpAdapter "github.com/datacenter/internal/simulator/tcp"
	"github.com/rs/zerolog"
)

var (
	protocol = flag.String("protocol", "", "Protocol type: mqtt, tcp, modbus, opcua")
	config   = flag.String("config", "", "Scenario config file path")
	mode     = flag.String("mode", "", "Mode: client/server for tcp, master/slave for modbus")
	devices  = flag.Int("devices", 0, "Override device count (0 = use config)")
	interval = flag.String("interval", "", "Override report interval")
	duration = flag.String("duration", "", "Override runtime duration (0 = infinite)")
	verbose  = flag.Bool("verbose", false, "Enable verbose logging")
)

func main() {
	flag.Parse()

	// Validate required flags
	if *protocol == "" {
		fmt.Fprintln(os.Stderr, "Error: --protocol is required")
		flag.Usage()
		os.Exit(1)
	}
	if *config == "" {
		fmt.Fprintln(os.Stderr, "Error: --config is required")
		flag.Usage()
		os.Exit(1)
	}

	// Setup logger
	logLevel := zerolog.InfoLevel
	if *verbose {
		logLevel = zerolog.DebugLevel
	}
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr}).
		Level(logLevel).
		With().
		Timestamp().
		Str("component", "simulator").
		Logger()

	// Load scenario config
	scenarioCfg, err := simulator.LoadScenarioConfig(*config)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to load scenario config")
	}

	// Load global config
	globalConfigPath := filepath.Join(filepath.Dir(*config), "..", "simulator.yaml")
	globalCfg, err := simulator.LoadGlobalConfig(globalConfigPath)
	if err != nil {
		logger.Warn().Err(err).Msg("Failed to load global config, using defaults")
		globalCfg = &simulator.GlobalConfig{}
	}

	// Load device templates
	templatesDir := filepath.Join(filepath.Dir(*config), "..", "devices")
	templates, err := simulator.LoadDeviceTemplates(templatesDir)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to load device templates")
	}

	// Create protocol adapter
	adapter, err := createAdapter(*protocol, &scenarioCfg.Scenario, globalCfg, logger)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create adapter")
	}

	// Create and start engine
	engine := simulator.NewSimulatorEngine(
		*protocol,
		&scenarioCfg.Scenario,
		templates,
		adapter,
		logger,
	)

	// Setup signal handling
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.Info().Str("signal", sig.String()).Msg("Received signal, shutting down")
		cancel()
	}()

	// Start engine
	if err := engine.Start(ctx); err != nil {
		logger.Fatal().Err(err).Msg("Failed to start engine")
	}

	// Wait for context cancellation
	<-ctx.Done()

	// Stop engine
	engine.Stop()
	logger.Info().Msg("Simulator exited")
}

func createAdapter(
	protocol string,
	scenario *simulator.ScenarioDefinition,
	globalCfg *simulator.GlobalConfig,
	logger zerolog.Logger,
) (simulator.ProtocolAdapter, error) {
	switch protocol {
	case "mqtt":
		return createMQTTAdapter(scenario, globalCfg, logger)
	case "tcp":
		return createTCPAdapter(scenario, globalCfg, logger, *mode)
	case "modbus":
		return createModbusAdapter(scenario, globalCfg, logger, *mode)
	case "opcua":
		return createOPCUAAdapter(scenario, globalCfg, logger)
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", protocol)
	}
}

func createMQTTAdapter(
	scenario *simulator.ScenarioDefinition,
	globalCfg *simulator.GlobalConfig,
	logger zerolog.Logger,
) (simulator.ProtocolAdapter, error) {
	// Get MQTT config from scenario or global
	broker := "localhost:1883"
	username := ""
	password := ""
	clientIDPrefix := "sim-"
	topicFormat := "devices/{domain_id}/{region}/{device_id}/telemetry"
	qos := byte(1)
	cleanSession := true

	if scenario.MQTT != nil {
		if scenario.MQTT.Broker != "" {
			broker = scenario.MQTT.Broker
		}
		if scenario.MQTT.Username != "" {
			username = scenario.MQTT.Username
		}
		if scenario.MQTT.Password != "" {
			password = scenario.MQTT.Password
		}
		if scenario.MQTT.ClientIDPrefix != "" {
			clientIDPrefix = scenario.MQTT.ClientIDPrefix
		}
		if scenario.MQTT.TopicFormat != "" {
			topicFormat = scenario.MQTT.TopicFormat
		}
		qos = scenario.MQTT.QoS
		cleanSession = scenario.MQTT.CleanSession
	} else if globalCfg.MQTT.Broker != "" {
		broker = globalCfg.MQTT.Broker
		if globalCfg.MQTT.Username != "" {
			username = globalCfg.MQTT.Username
		}
		if globalCfg.MQTT.Password != "" {
			password = globalCfg.MQTT.Password
		}
		if globalCfg.MQTT.ClientIDPrefix != "" {
			clientIDPrefix = globalCfg.MQTT.ClientIDPrefix
		}
	}

	return mqttAdapter.NewAdapter(
		broker,
		username,
		password,
		clientIDPrefix,
		topicFormat,
		qos,
		cleanSession,
		logger,
	), nil
}

func createTCPAdapter(
	scenario *simulator.ScenarioDefinition,
	globalCfg *simulator.GlobalConfig,
	logger zerolog.Logger,
	mode string,
) (simulator.ProtocolAdapter, error) {
	// Get TCP config from scenario or global
	host := "localhost"
	port := 9000

	if scenario.TCP != nil {
		if scenario.TCP.Host != "" {
			host = scenario.TCP.Host
		}
		if scenario.TCP.Port > 0 {
			port = scenario.TCP.Port
		}
		if scenario.TCP.Mode != "" && mode == "" {
			mode = scenario.TCP.Mode
		}
	} else if globalCfg.TCP.Host != "" {
		host = globalCfg.TCP.Host
		if globalCfg.TCP.Port > 0 {
			port = globalCfg.TCP.Port
		}
	}

	// Default mode
	if mode == "" {
		mode = "client"
	}

	return tcpAdapter.NewAdapter(host, port, mode, logger), nil
}

func createModbusAdapter(
	scenario *simulator.ScenarioDefinition,
	globalCfg *simulator.GlobalConfig,
	logger zerolog.Logger,
	mode string,
) (simulator.ProtocolAdapter, error) {
	// Get Modbus config from scenario or global
	host := "localhost"
	port := 502
	slaveIDs := []byte{1, 2, 3, 4, 5}

	if scenario.Modbus != nil {
		if scenario.Modbus.Host != "" {
			host = scenario.Modbus.Host
		}
		if scenario.Modbus.Port > 0 {
			port = scenario.Modbus.Port
		}
		if scenario.Modbus.Mode != "" && mode == "" {
			mode = scenario.Modbus.Mode
		}
		if len(scenario.Modbus.SlaveIDs) > 0 {
			slaveIDs = scenario.Modbus.SlaveIDs
		}
	} else if globalCfg.Modbus.Host != "" {
		host = globalCfg.Modbus.Host
		if globalCfg.Modbus.Port > 0 {
			port = globalCfg.Modbus.Port
		}
	}

	// Default mode
	if mode == "" {
		mode = "slave"
	}

	return modbusAdapter.NewAdapter(host, port, mode, slaveIDs, logger), nil
}

func createOPCUAAdapter(
	scenario *simulator.ScenarioDefinition,
	globalCfg *simulator.GlobalConfig,
	logger zerolog.Logger,
) (simulator.ProtocolAdapter, error) {
	// Get OPC UA config from scenario or global
	endpoint := "opc.tcp://localhost:4840"
	pollInterval := 5 * time.Second
	nodeIDs := []string{
		"ns=2;s=Motor1.Speed",
		"ns=2;s=Motor1.Current",
		"ns=2;s=Temp1.Value",
	}

	if scenario.OPCUA != nil {
		if scenario.OPCUA.Endpoint != "" {
			endpoint = scenario.OPCUA.Endpoint
		}
		if scenario.OPCUA.PollInterval != "" {
			d, err := time.ParseDuration(scenario.OPCUA.PollInterval)
			if err == nil {
				pollInterval = d
			}
		}
		if len(scenario.OPCUA.NodeIDs) > 0 {
			nodeIDs = scenario.OPCUA.NodeIDs
		}
	} else if globalCfg.OPCUA.Endpoint != "" {
		endpoint = globalCfg.OPCUA.Endpoint
	}

	return opcuaAdapter.NewAdapter(endpoint, pollInterval, nodeIDs, logger), nil
}
