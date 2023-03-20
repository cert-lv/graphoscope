package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"plugin"
	"reflect"
	"time"

	yaml "gopkg.in/yaml.v3"

	"github.com/cert-lv/graphoscope/pdk"
)

var (
	// Loaded plugins
	plugins map[string]interface{}

	// Collectors for the preconfigured data sources.
	// Is a map of data source's name -> related plugin
	collectors map[string]pdk.SourcePlugin

	// Processors of the data received by the collectors,
	// runs in a background for each entry
	processors = []pdk.ProcessorPlugin{}

	// In a Web GUI do not display some elements
	// when there are no non-global data sources
	nonGlobalExist = false

	// A list of all known data sources fields
	// for the Web GUI autocomplete
	fields map[string][]string
)

/*
 * Load plugins from a configured directory
 */
func loadPlugins() error {
	plugins = make(map[string]interface{})

	// Load several types of plugins
	for _, group := range []string{"sources", "processors", "outputs"} {

		files, err := ioutil.ReadDir(config.Plugins + "/" + group)
		if err != nil {
			return fmt.Errorf("Can't read from '%s' directory: %s", config.Plugins+"/"+group, err.Error())
		}

		for _, f := range files {
			// Skip non-plugin files
			name := f.Name()
			if name[len(name)-3:] != ".so" {
				continue
			}

			// Open a .so file to load the symbols
			plug, err := plugin.Open(config.Plugins + "/" + group + "/" + name)
			if err != nil {
				return fmt.Errorf("Can't open '%s': %s", name, err.Error())
			}

			// Look up the main plugin's symbol
			symPlugin, err := plug.Lookup("Plugin")
			if err != nil {
				return fmt.Errorf("Can't lookup symbol 'Plugin' in '%s': %s", name, err.Error())
			}

			// Get plugin name
			symName, err := plug.Lookup("Name")
			if err != nil {
				return fmt.Errorf("Can't lookup symbol 'Name' in '%s': %s", name, err.Error())
			}
			pName, ok := symName.(*string)
			if !ok {
				return fmt.Errorf("Unexpected plugin name type in '%s': %T, '*string' expected", name, pName)
			}

			// Get plugin version
			symVersion, err := plug.Lookup("Version")
			if err != nil {
				return fmt.Errorf("Can't lookup symbol 'Version' in '%s': %s", name, err.Error())
			}
			pVersion, ok := symVersion.(*string)
			if !ok {
				return fmt.Errorf("Unexpected plugin version type in '%s': %T, '*string' expected", name, pVersion)
			}

			// Assert that loaded symbol is of a desired type
			// and make plugin globally available
			// switch group {
			// case "source":
			// 	plugin, ok := symPlugin.(pdk.SourcePlugin)
			// 	plugins[*pName] = plugin
			// case "process":
			// 	plugin, ok := symPlugin.(pdk.ProcessPlugin)
			// 	plugins[*pName] = plugin
			// case "output":
			// 	plugin, ok := symPlugin.(pdk.OutputPlugin)
			// 	plugins[*pName] = plugin
			// }

			// if !ok {
			// 	return fmt.Errorf("Invalid plugin's type of '%s': %T, all methods must be implemented", name, symPlugin)
			// }

			plugins[*pName] = symPlugin

			log.Info().
				Str("plugin", *pName).
				Msg("Plugin loaded, version " + *pVersion)
		}
	}

	return nil
}

/*
 * Setup collectors for the predefined data sources
 */
func setupCollectors() error {
	// Clear old content
	collectors = make(map[string]pdk.SourcePlugin)

	files, err := ioutil.ReadDir(config.Definitions + "/sources")
	if err != nil {
		return fmt.Errorf("Can't read directory '%s': %s", config.Definitions+"/sources", err.Error())
	}

	// A map of data sources fields,
	// source name -> list
	fields = make(map[string][]string)

	// Reset flag in case collectors are reloaded without service restart
	nonGlobalExist = false

	for _, f := range files {
		// Skip not YAML files
		name := f.Name()
		if len(name) <= 5 || name[len(name)-5:] != ".yaml" {
			continue
		}

		def, err := loadSource(config.Definitions + "/sources/" + name)
		if err != nil {
			log.Error().Msgf("Can't load source file '%s': %s", name, err.Error())
			continue
		}

		// Use needed plugin
		collector, ok := plugins[def.Plugin].(pdk.SourcePlugin)
		if !ok {
			log.Error().
				Str("source", def.Name).
				Str("plugin", def.Plugin).
				Msg("No such plugin required by a collector")
			continue
		}

		// Clone interface to avoid pointers in "collectors" to the same value
		collectorIntf := reflect.New(reflect.TypeOf(collector).Elem())
		clone := collectorIntf.Interface().(pdk.SourcePlugin)

		// Close previous connection if exists
		err = clone.Stop()
		if err != nil {
			log.Error().
				Str("source", def.Name).
				Str("plugin", def.Plugin).
				Msg("Can't stop collector: " + err.Error())
			continue
		}

		// Set current unique parameters
		err = clone.Setup(def, config.Limit)
		if err != nil {
			log.Error().
				Str("source", def.Name).
				Str("plugin", def.Plugin).
				Msg("Can't setup: " + err.Error())
			continue
		}

		// Get all the possible field names
		list, err := clone.Fields()
		if err != nil {
			log.Error().
				Str("source", def.Name).
				Str("plugin", def.Plugin).
				Msg("Can't get fields: " + err.Error())
		}

		// Prevent NULL values in resulting JSON
		if len(list) == 0 {
			list = make([]string, 0)
		}

		// Rename common field names
		for renamed, old := range clone.Conf().ReplaceFields {
			for i, field := range list {
				if field == old {
					list[i] = renamed
					break
				}
			}
		}

		// Merge field names with a global list
		fields[def.Name] = list

		if !clone.Conf().InGlobal {
			nonGlobalExist = true
		}

		// Store collectors to be usable by the end-users
		collectors[def.Name] = clone

		log.Info().
			Str("source", def.Name).
			Str("plugin", def.Plugin).
			Msg("Collector initialized")
	}

	return nil
}

/*
 * Setup processors of the data sources received data
 */
func setupProcessors() error {
	// Clear old content
	processors = []pdk.ProcessorPlugin{}

	files, err := ioutil.ReadDir(config.Definitions + "/processors")
	if err != nil {
		return fmt.Errorf("Can't read directory '%s': %s", config.Definitions+"/processors", err.Error())
	}

	for _, f := range files {
		// Skip not YAML files
		name := f.Name()
		if len(name) <= 5 || name[len(name)-5:] != ".yaml" {
			continue
		}

		def, err := loadProcessor(config.Definitions + "/processors/" + name)
		if err != nil {
			log.Error().Msgf("Can't load processor file '%s': %s", name, err.Error())
			continue
		}

		// Use needed plugin
		processor, ok := plugins[def.Plugin].(pdk.ProcessorPlugin)
		if !ok {
			log.Error().
				Str("process", def.Name).
				Str("plugin", def.Plugin).
				Msg("No such plugin required by a processor")
			continue
		}

		// Clone interface to avoid pointers in "processors" to the same value
		processorIntf := reflect.New(reflect.TypeOf(processor).Elem())
		clone := processorIntf.Interface().(pdk.ProcessorPlugin)

		// Close previous connection if exists
		err = clone.Stop()
		if err != nil {
			log.Error().
				Str("process", def.Name).
				Str("plugin", def.Plugin).
				Msg("Can't stop processor: " + err.Error())
			continue
		}

		// Set current unique parameters
		err = clone.Setup(def)
		if err != nil {
			log.Error().
				Str("process", def.Name).
				Str("plugin", def.Plugin).
				Msg("Can't setup: " + err.Error())
			continue
		}

		// Store processors to be usable by the end-users
		processors = append(processors, clone)

		log.Info().
			Str("processor", def.Name).
			Str("plugin", def.Plugin).
			Msg("Processor initialized")
	}

	return nil
}

/*
 * Load data source configuration file
 */
func loadSource(filename string) (*pdk.Source, error) {
	confFile, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("Can't open: " + err.Error())
	}

	fi, _ := confFile.Stat()
	buffer := make([]byte, fi.Size())
	_, err = confFile.Read(buffer)
	if err != nil {
		return nil, fmt.Errorf("Can't read: " + err.Error())
	}

	source := &pdk.Source{}
	err = yaml.Unmarshal(buffer, &source)
	if err != nil {
		return nil, fmt.Errorf("Can't unmarshall: " + err.Error())
	}

	// Set default values if not specified
	if source.Timeout == 0*time.Second {
		source.Timeout = 60 * time.Second
	}

	return source, nil
}

/*
 * Load processor configuration file
 */
func loadProcessor(filename string) (*pdk.Processor, error) {
	confFile, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("Can't open: " + err.Error())
	}

	fi, _ := confFile.Stat()
	buffer := make([]byte, fi.Size())
	_, err = confFile.Read(buffer)
	if err != nil {
		return nil, fmt.Errorf("Can't read: " + err.Error())
	}

	processor := &pdk.Processor{}
	err = yaml.Unmarshal(buffer, &processor)
	if err != nil {
		return nil, fmt.Errorf("Can't unmarshall: " + err.Error())
	}

	// Set default values if not specified
	if processor.Timeout == 0*time.Second {
		processor.Timeout = 60 * time.Second
	}

	return processor, nil
}
