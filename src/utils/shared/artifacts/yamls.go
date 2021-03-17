package artifacts

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"pixielabs.ai/pixielabs/src/cloud/cloudapipb"
	"pixielabs.ai/pixielabs/src/utils/shared/tar"
	"pixielabs.ai/pixielabs/src/utils/shared/yamls"
)

func downloadFile(url string) (io.ReadCloser, error) {
	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	return resp.Body, nil
}

func downloadVizierYAMLs(conn *grpc.ClientConn, authToken, versionStr string, templated bool) (io.ReadCloser, error) {
	client := cloudapipb.NewArtifactTrackerClient(conn)
	at := cloudapipb.AT_CONTAINER_SET_YAMLS
	if templated {
		at = cloudapipb.AT_CONTAINER_SET_TEMPLATE_YAMLS
	}

	req := &cloudapipb.GetDownloadLinkRequest{
		ArtifactName: "vizier",
		VersionStr:   versionStr,
		ArtifactType: at,
	}
	ctxWithCreds := metadata.AppendToOutgoingContext(context.Background(), "authorization",
		fmt.Sprintf("bearer %s", authToken))

	resp, err := client.GetDownloadLink(ctxWithCreds, req)
	if err != nil {
		return nil, err
	}

	return downloadFile(resp.Url)
}

// FetchVizierYAMLMap fetches Vizier YAML files and write to a map <fname>:<yaml string>.
func FetchVizierYAMLMap(conn *grpc.ClientConn, authToken, versionStr string) (map[string]string, error) {
	reader, err := downloadVizierYAMLs(conn, authToken, versionStr, false)

	if err != nil {
		return nil, err
	}
	defer reader.Close()

	yamlMap, err := tar.ReadTarFileFromReader(reader)
	if err != nil {
		return nil, err
	}
	return yamlMap, nil
}

// FetchVizierTemplates fetches the Vizier templates for the given version.
func FetchVizierTemplates(conn *grpc.ClientConn, authToken, versionStr string) ([]*yamls.YAMLFile, error) {
	reader, err := downloadVizierYAMLs(conn, authToken, versionStr, true)

	if err != nil {
		return nil, err
	}
	defer reader.Close()

	yamlMap, err := tar.ReadTarFileFromReader(reader)
	if err != nil {
		return nil, err
	}

	// Convert to YAML files, using the provided file names.
	// Get the YAML names, in order.
	yamlNames := make([]string, len(yamlMap))
	i := 0
	for k := range yamlMap {
		yamlNames[i] = k
		i++
	}
	sort.Strings(yamlNames)

	// Write to YAMLFile slice.
	var yamlFiles []*yamls.YAMLFile
	re := regexp.MustCompile(`(?:[0-9]+_)(.*)(?:\.yaml)`)
	for _, fName := range yamlNames {
		// The filename looks like "./pixie_yamls/00_namespace.yaml", we want to extract the "namespace".
		ms := re.FindStringSubmatch(fName)
		if ms == nil || len(ms) != 2 {
			continue
		}
		yamlFiles = append(yamlFiles, &yamls.YAMLFile{
			Name: ms[1],
			YAML: yamlMap[fName],
		})
	}

	return yamlFiles, nil
}