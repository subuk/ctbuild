package main

import (
	"archive/tar"
	"bytes"
	"github.com/hashicorp/hcl"
	"io"
	"io/ioutil"
)

type BuildEnvConfigFile struct {
	Name    string `hcl:",key"`
	Mode    int64  `hcl:"mode"`
	Content string `hcl:"content"`
}

type BuildEnvConfig struct {
	Name      string
	BaseImage string               `hcl:"base_image"`
	BuildCmd  string               `hcl:"build_cmd"`
	Files     []BuildEnvConfigFile `hcl:"file"`
}

func ParseBuildEnvConfig(filename string) (*BuildEnvConfig, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	config := &BuildEnvConfig{}
	if err := hcl.Unmarshal(content, config); err != nil {
		return nil, err
	}
	return config, nil
}

func (b *BuildEnvConfig) FilesArchive() (io.Reader, error) {
	buf := new(bytes.Buffer)
	tw := tar.NewWriter(buf)
	for _, f := range b.Files {
		hdr := &tar.Header{
			Name: f.Name,
			Mode: f.Mode,
			Size: int64(len(f.Content)),
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return nil, Error{err, "failed to write tar header"}
		}
		if _, err := io.WriteString(tw, f.Content); err != nil {
			return nil, Error{err, "failed to write tar file content"}
		}
	}
	if err := tw.Close(); err != nil {
		return nil, Error{err, "failed to close tar file"}
	}

	return bytes.NewReader(buf.Bytes()), nil
}
