package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"plugin"
	"reflect"
	"sort"
	"time"

	yaml "gopkg.in/yaml.v3"

	"github.com/cert-lv/graphoscope/pdk"
)

var (
	// Loaded plugins.
	// "pdk.Plugin" struct is an interface,
	// so the next line is a map of pointers
	plugins map[string]pdk.Plugin

	// Collectors for the preconfigured data sources.
	// Is a map of data source's name -> related plugin
	collectors map[string]pdk.Plugin

	// In a Web GUI do not display some elements
	// when there are no non-global data sources
	nonGlobalExist = false

	// A list of all known data sources fields
	// for the Web GUI autocomplete
	fields []string
)

/*
 * Load plugins from a configured directory
 */
func loadPlugins() error {
	plugins = make(map[string]pdk.Plugin)

	files, err := ioutil.ReadDir(config.Plugins)
	if err != nil {
		return fmt.Errorf("Can't read from '%s' directory: %s", config.Plugins, err.Error())
	}

	for _, f := range files {
		// Skip non-plugin files
		name := f.Name()
		if name[len(name)-3:] != ".so" {
			continue
		}

		// Open a .so file to load the symbols
		plug, err := plugin.Open(config.Plugins + "/" + name)
		if err != nil {
			return fmt.Errorf("Can't open '%s': %s", name, err.Error())
		}

		// Look up the main plugin's symbol
		symPlugin, err := plug.Lookup("Plugin")
		if err != nil {
			return fmt.Errorf("Can't lookup symbol 'Plugin' in '%s': %s", name, err.Error())
		}

		// Assert that loaded symbol is of a desired type
		plugin, ok := symPlugin.(pdk.Plugin)
		if !ok {
			return fmt.Errorf("Invalid plugin's type in '%s': %T, '*pdk.Plugin' expected with all methods implemented", name, plugin)
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

		// Make plugin globally available
		plugins[*pName] = plugin

		log.Info().
			Str("plugin", *pName).
			Msg("Plugin loaded, version " + *pVersion)
	}

	return nil
}

/*
 * Setup collectors for the predefined data sources
 */
func setupCollectors() error {
	collectors = make(map[string]pdk.Plugin)

	files, err := ioutil.ReadDir(config.Sources)
	if err != nil {
		return fmt.Errorf("Can't read directory '%s': %s", config.Sources, err.Error())
	}

	// Map of unique fields
	uniqueFields := map[string]bool{}

	for _, f := range files {
		// Skip not YAML files
		name := f.Name()
		if len(name) <= 5 || name[len(name)-5:] != ".yaml" {
			continue
		}

		source, err := loadSource(config.Sources + "/" + name)
		if err != nil {
			log.Error().Msgf("Can't load source file '%s': %s", name, err.Error())
			continue
		}

		// Use needed plugin
		collector, ok := plugins[source.Plugin]
		if !ok {
			log.Error().
				Str("source", source.Name).
				Str("plugin", source.Plugin).
				Msg("No such plugin required by a collector")
			continue
		}

		// Clone interface to avoid pointers in "collectors" to the same value
		collectorIntf := reflect.New(reflect.TypeOf(collector).Elem())
		clone := collectorIntf.Interface().(pdk.Plugin)

		// Set current unique parameters
		err = clone.Setup(source, config.Limit)
		if err != nil {
			log.Error().
				Str("source", source.Name).
				Str("plugin", source.Plugin).
				Msg("Can't setup: " + err.Error())
			continue
		}

		// Get all the possible field names
		list, err := clone.Fields()
		if err != nil {
			log.Error().
				Str("source", source.Name).
				Str("plugin", source.Plugin).
				Msg("Can't get fields: " + err.Error())
		}

		// Rename common field names
		for renamed, old := range clone.Source().ReplaceFields {
			for i, field := range list {
				if field == old {
					list[i] = renamed
					break
				}
			}
		}

		// Merge field names with a global list
		for _, field := range list {
			uniqueFields[field] = true
		}

		if !clone.Source().InGlobal {
			nonGlobalExist = true
		}

		// Store collectors to be usable by the end-users
		collectors[source.Name] = clone

		log.Info().
			Str("source", source.Name).
			Str("plugin", source.Plugin).
			Msg("Collector initialized")
	}

	// Create a slice with the capacity of unique fields.
	// This capacity makes appending flow much more efficient
	fields = make([]string, 0, len(uniqueFields))

	for field := range uniqueFields {
		fields = append(fields, field)
	}

	// Sort keys for easier usage
	sort.Strings(fields)

	return nil
}

/*
 * Load data source definition file
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
