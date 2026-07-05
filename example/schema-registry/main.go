// schema-registry is a small verification program that uses the kafui
// datasource implementation to fetch schemas from the Confluent Schema
// Registry configured in ~/.kaf/config.
//
// Run with:
//
//	go run ./example/schema-registry
//
// The program reads the active cluster from the default kaf config file,
// prints the cluster name and registry URL, then lists every registered
// schema subject together with its latest version, schema ID, and type.
package main

import (
	"fmt"
	"os"

	"github.com/Benny93/kafui/pkg/datasource/kafds"
)

func main() {
	// Load ~/.kaf/config and set the active cluster (same as the CLI does
	// via cobra.OnInitialize).
	if err := kafds.InitFromConfig(""); err != nil {
		fmt.Fprintf(os.Stderr, "❌  Failed to load kaf config: %v\n", err)
		fmt.Fprintf(os.Stderr, "    Make sure ~/.kaf/config exists and is readable.\n")
		os.Exit(1)
	}

	ds := kafds.NewKafkaDataSourceKaf()

	// Show active cluster name and its schema registry URL.
	activeCtx := ds.GetContext()
	fmt.Printf("Active cluster: %s\n", activeCtx)

	clusterDetails, err := ds.GetClusterDetails(activeCtx)
	if err == nil && clusterDetails.SchemaRegistryURL != "" {
		fmt.Printf("Schema Registry URL: %s\n\n", clusterDetails.SchemaRegistryURL)
	} else {
		fmt.Println("ℹ️  No Schema Registry URL found for the active cluster.")
		fmt.Println("   Set schema-registry-url in ~/.kaf/config under the active cluster.")
		fmt.Println()
	}

	fmt.Println("Fetching schemas …")
	schemas, err := ds.GetSchemas()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌  GetSchemas failed: %v\n", err)
		os.Exit(1)
	}

	if len(schemas) == 0 {
		fmt.Println("No schemas found. The registry may be empty or unreachable.")
		return
	}

	fmt.Printf("✅  Found %d subject(s). Fetching details for first page…\n\n", len(schemas))

	// Fetch details for the first 50 subjects only (simulating one page).
	const pageSize = 50
	end := len(schemas)
	if end > pageSize {
		end = pageSize
	}
	subjects := make([]string, end)
	for i, s := range schemas[:end] {
		subjects[i] = s.Subject
	}
	details, err := ds.GetSchemaDetails(subjects)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌  GetSchemaDetails failed: %v\n", err)
		os.Exit(1)
	}

	// Index details by subject for display.
	bySubject := make(map[string]struct {
		Version    int
		ID         int
		SchemaType string
	}, len(details))
	for _, d := range details {
		schemaType := d.SchemaType
		if schemaType == "" {
			schemaType = "AVRO"
		}
		bySubject[d.Subject] = struct {
			Version    int
			ID         int
			SchemaType string
		}{d.Version, d.ID, schemaType}
	}

	fmt.Printf("  %-50s  %7s  %7s  %s\n", "Subject", "Version", "ID", "Type")
	fmt.Printf("  %-50s  %7s  %7s  %s\n",
		"--------------------------------------------------",
		"-------", "-------", "-------")
	for _, s := range schemas[:end] {
		d := bySubject[s.Subject]
		fmt.Printf("  %-50s  %7d  %7d  %s\n", s.Subject, d.Version, d.ID, d.SchemaType)
	}
	if len(schemas) > pageSize {
		fmt.Printf("\n  … and %d more subjects (only first page shown)\n", len(schemas)-pageSize)
	}
}
