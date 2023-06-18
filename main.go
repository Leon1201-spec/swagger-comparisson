package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Name    string   `yaml:"name"`
	Path    string   `yaml:"path"`
	Https   bool     `yaml:"https"`
	Hosts   []string `yaml:"hosts"`
	Webhook string   `yaml:"slack-webhook"`
	Channel string   `yaml:"slack-channel"`
}

type Difference struct {
	Path    string
	Value   interface{}
	Parents []string
	Type    string
}

func main() {
	// Read the yaml file
	yamlFile, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		log.Fatalf("Error reading YAML file : %v\n", err)
	}

	// Unmarshal yaml file to Config Struct passed above
	var config Config
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		log.Fatalf("Error parsing YAML: %v\n", err)
	}

	// Configuring string for Swagger JSON endpoint
	hosts := []string{}
	addString := config.Path
	if config.Https == true {
		protol := "https://"
		for i := range config.Hosts {
			x := protol + config.Hosts[i] + addString
			hosts = append(hosts, x)
		}
	} else {
		protol := "http://"
		for i := range config.Hosts {
			x := protol + config.Hosts[i] + addString
			hosts = append(hosts, x)
		}
	}

	for i := range hosts {
		latest := get_last_num(config.Hosts[i])
		fileNames := get_swagger(hosts[i], config.Hosts[i], latest)
		if latest < 1 {
			fmt.Println("Nothing to compare for: ", config.Hosts[i])
		} else {
			return_string, changes := compare_json(fileNames[0], fileNames[1], config.Hosts[i])
			slack_notification(return_string, changes, config.Webhook, config.Channel)
			//fmt.Println(return_string, changes)
		}
	}
}

func get_last_num(name string) int {
	// Regular expression to match filenames with numbers
	regex := regexp.MustCompile(`(\d+)`)

	// Read the filenames from the directory
	files, err := ioutil.ReadDir(name)
	if err != nil {
		fmt.Printf("Error reading directory: %v\n", err)
		highestNumber := 0
		return highestNumber
	} else {
		highestNumber := 0

		// Iterate over the files
		for _, file := range files {
			// Extract the number from the filename
			matches := regex.FindStringSubmatch(file.Name())
			if len(matches) > 1 {
				number, _ := strconv.Atoi(matches[1])
				if number > highestNumber {
					highestNumber = number
				}
			}
		}
		return highestNumber
	}
}

func get_swagger(host string, name string, latest int) []string {

	// Send get request
	response, err := http.Get(host)
	if err != nil {
		log.Fatalf("Error sending GET request: %v\n", err)
	}
	defer response.Body.Close()

	// Create folder if doesnt exist
	err_folder := os.MkdirAll(name, os.ModePerm)
	if err_folder != nil {
		fmt.Printf("Error creating folder: %v\n", err)
	}

	// Create File naming and return list
	latest_str_old := strconv.Itoa(latest)
	latest_str_new := strconv.Itoa(latest + 1)
	fileName_old := latest_str_old + ".json"
	fileName_new := latest_str_new + ".json"
	filePath_new := name + "/" + fileName_new
	filePath_old := name + "/" + fileName_old
	fileNames := []string{}
	fileNames = append(fileNames, filePath_old)
	fileNames = append(fileNames, filePath_new)

	// Create output file
	file, err := os.Create(filePath_new)
	if err != nil {
		log.Fatalf("Error creating output file: %v\n", err)
	}
	defer file.Close()

	// Copy response body to file
	_, err = io.Copy(file, response.Body)
	if err != nil {
		fmt.Printf("Error copying response body to file: %v\n", err)
	}
	return fileNames
}

func compare_json(file_old string, file_new string, name string) (string, []string) {

	// Reading File 1 + Error Logging
	file1, err := ioutil.ReadFile(file_old)
	if err != nil {
		log.Fatalf("Erros while readng in file %v: %v\n", file_old, err)
	}

	// Reading File 2 + Error Logging
	file2, err := ioutil.ReadFile(file_new)
	if err != nil {
		log.Fatalf("Erros while readng in file %v: %v\n", file_new, err)
	}

	// Unmarshal Json file 1 to data1
	var data1 interface{}
	if err := json.Unmarshal(file1, &data1); err != nil {
		log.Fatalf("Error parsing JSON in file %v: %v\n", file_old, err)
	}

	// Unmarshal Json file 2 to data2
	var data2 interface{}
	if err := json.Unmarshal(file2, &data2); err != nil {
		log.Fatalf("Error parsing JSON in file %v: %v\n", file_new, err)
	}

	// Compare the swagger files
	differences, isEqual := compareSwagger(data1, data2, []string{})
	changes := []string{}

	if isEqual {
		return_string := "JSON objects are equal for: " + name
		return return_string, changes
	}
	return_string := "JSON objects are not equal. Differences: " + name + "\n"

	for _, diff := range differences {

		// Defining Change Return
		path := strings.TrimPrefix(diff.Path, ".")
		change_value := fmt.Sprintf("%v", diff.Value)
		parent := fmt.Sprint(diff.Parents)
		change := diff.Type + " " + path + " Changed Value: " + change_value + " " + parent + " Changed Endpoints:"
		changes = append(changes, change)

		// Returning Enpoints that are affected by this model
		start_list := []string{}
		fixedPath := updatePath(parent)
		start_list = append(start_list, fixedPath)

		finalEndpoints := []string{}
		endpointsLoop(data2, start_list, path, &finalEndpoints)
		boldList := make([]string, len(finalEndpoints))
		for i := range finalEndpoints {
			boldList[i] = "*" + finalEndpoints[i] + "*"
		}
		changes = append(changes, boldList...)
	}
	return return_string, changes
}

func compareSwagger(data1, data2 interface{}, parents []string) ([]Difference, bool) {
	differences := make([]Difference, 0)

	// Compare Datatypes
	if fmt.Sprintf("%T", data1) != fmt.Sprintf("%T", data2) {
		fmt.Println("The datatypes are not the same")
		return differences, false
	}

	// Compare the values of the two objects based on their types
	switch d1 := data1.(type) {
	case map[string]interface{}:
		d2, ok := data2.(map[string]interface{})
		if !ok {
			// data2 is not a map, so all keys in data1 are considered as deletions
			for k := range d1 {
				differences = append(differences, Difference{Path: k, Value: d1[k], Parents: parents, Type: "Deletion"})
			}
			return differences, false
		}

		for k, v1 := range d1 {
			if v2, ok := d2[k]; !ok {
				// key is deleted
				differences = append(differences, Difference{Path: k, Value: v1, Parents: parents, Type: "Deletion"})
			} else {
				subParents := append(parents, k)
				subDifferences, isEqual := compareSwagger(v1, v2, subParents)
				if !isEqual {
					differences = append(differences, subDifferences...)
				}
			}
		}

		// Check for keys in data2 that are not present in data1 (added lines)
		for k, v2 := range d2 {
			if _, ok := d1[k]; !ok {
				// key is added
				differences = append(differences, Difference{Path: k, Value: v2, Parents: parents, Type: "Addition"})
			}
		}

		return differences, len(differences) == 0

	case []interface{}:
		d2, ok := data2.([]interface{})
		if !ok || len(d1) != len(d2) {
			differences = append(differences, Difference{Path: "", Value: nil, Parents: parents})
			return differences, false
		}
		for i, v1 := range d1 {
			subParents := append(parents, fmt.Sprintf("[%d]", i))
			subDifferences, isEqual := compareSwagger(v1, d2[i], subParents)
			if !isEqual {
				differences = append(differences, subDifferences...)
			}
		}
		return differences, len(differences) == 0

	default:
		if data1 != data2 {
			differences = append(differences, Difference{Path: "", Value: data1, Parents: parents, Type: "Modification"})
		}
		return differences, len(differences) == 0
	}
}

func get_endpoint(data interface{}, targetValue string, path string, endpoints []string) []string {
	switch value := data.(type) {
	case map[string]interface{}:
		for key, v := range value {
			newPath := fmt.Sprintf("%s.%s", path, key)
			endpoints = get_endpoint(v, targetValue, newPath, endpoints)
		}
	case []interface{}:
		for i, v := range value {
			newPath := fmt.Sprintf("%s[%d]", path, i)
			endpoints = get_endpoint(v, targetValue, newPath, endpoints)
		}
	default:
		if fmt.Sprintf("%v", value) == targetValue {
			parentEndpoint := extractPath(path)
			endpoints = append(endpoints, parentEndpoint)
		}
	}

	return endpoints
}

func endpointsLoop(data interface{}, endpoints []string, path string, finalEndpoints *[]string) {
	for _, value := range endpoints {
		loop := get_endpoint(data, value, path, []string{})
		if len(loop) == 0 {
			*finalEndpoints = append(*finalEndpoints, value)
		} else {
			uniqueEndpoints := removeDuplicatesFromSlice(loop)
			endpointsLoop(data, uniqueEndpoints, path, finalEndpoints)
		}
	}
}

func extractPath(ref string) string {
	// Split the string by "."
	parts := strings.Split(ref, ".")

	// Remove unnecessay elements
	parts = parts[:3]

	// Join the parts back together with "/"
	path := strings.Join(parts, "/")

	if strings.HasPrefix(path, "fileUrls") || strings.HasPrefix(path, "/fileUrls") {
		path = strings.TrimPrefix(path, "fileUrls")
	} else if strings.HasPrefix(path, "/paths") || strings.HasPrefix(path, "paths") {
		path = strings.TrimPrefix(path, "paths")
	}

	if strings.HasPrefix(path, "/") {
		path = "#" + path
	} else {
		path = "#/" + path
	}
	// Prepend "#" to the path
	return path
}

func updatePath(ref string) string {
	ref = strings.TrimLeft(ref, "[")
	parts := strings.Split(ref, " ")

	// Remove unnecessary elements
	if len(parts) > 1 {
		parts = parts[:2]
	}

	// Join the parts back together with "/"
	path := strings.Join(parts, "/")

	// Prepend "#" to the path
	path = "#/" + path

	return path
}

func removeDuplicatesFromSlice(slice []string) []string {
	uniqueMap := make(map[string]bool)
	uniqueSlice := make([]string, 0)

	for _, str := range slice {
		if !uniqueMap[str] {
			uniqueMap[str] = true
			uniqueSlice = append(uniqueSlice, str)
		}
	}

	return uniqueSlice
}

func slack_notification(return_string string, changes []string, webhook string, channel string) {

	// List to String
	changes_str := strings.Join(changes, "\n")
	answere := return_string + " " + changes_str

	// Create the message payload
	message := map[string]interface{}{
		"channel": channel,
		"text":    answere,
	}

	// Convert the message payload to JSON
	payload, err := json.Marshal(message)
	if err != nil {
		log.Fatalf("Error marshaling JSON: %v\n", err)
		return
	}

	// Send the HTTP POST request to the Slack webhook URL
	resp, err := http.Post(webhook, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		log.Fatalf("Error sending Slack notification: %v\n", err)
		return
	}
	defer resp.Body.Close()
}
