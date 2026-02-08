package output

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/goccy/go-yaml"
)

// PrintJSON writes data as indented JSON to stdout
func PrintJSON(data interface{}) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// PrintYAML writes data as YAML to stdout
func PrintYAML(data interface{}) error {
	// Marshal through JSON first to respect json tags,
	// then unmarshal into a generic structure for clean YAML output
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	var generic interface{}
	if err := json.Unmarshal(jsonBytes, &generic); err != nil {
		return err
	}
	out, err := yaml.Marshal(generic)
	if err != nil {
		return err
	}
	_, err = os.Stdout.Write(out)
	return err
}

// PrintTable writes tabular data to stdout
func PrintTable(headers []string, rows [][]string) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

	// Print headers
	for i, h := range headers {
		if i > 0 {
			fmt.Fprint(w, "\t")
		}
		fmt.Fprint(w, h)
	}
	fmt.Fprintln(w)

	// Print rows
	for _, row := range rows {
		for i, col := range row {
			if i > 0 {
				fmt.Fprint(w, "\t")
			}
			fmt.Fprint(w, col)
		}
		fmt.Fprintln(w)
	}

	w.Flush()
}

// PrintFormatted handles the json/yaml/table output format switch.
// For "json" and "yaml" formats it serializes data directly.
// For any other format (typically "table") it calls tableFn.
func PrintFormatted(format string, data interface{}, tableFn func() error) error {
	switch format {
	case "json":
		return PrintJSON(data)
	case "yaml":
		return PrintYAML(data)
	default:
		return tableFn()
	}
}

// PrintMessage writes a plain message to stdout
func PrintMessage(msg string) {
	fmt.Println(msg)
}

// PrintError writes an error message to stderr
func PrintError(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
}
