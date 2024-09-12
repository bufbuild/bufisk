// Copyright 2023 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"

	"aead.dev/minisign"
)

const (
	// If not set, falls back to $XDG_CACHE_HOME and then $HOME/.cache
	// on unix, or %LocalAppData% for windows.
	cacheDirPathEnvKey  = "BUFISK_CACHE_DIR"
	useBufVersionEnvKey = "BUF_VERSION"
	bufVersionFileName  = ".bufversion"

	minisignPublicKey = "RWQ/i9xseZwBVE7pEniCNjlNOeeyp4BQgdZDLQcAohxEAH5Uj5DEKjv6"
)

var (
	interruptSignals = append(
		[]os.Signal{
			os.Interrupt,
		},
		extraInterruptSignals...,
	)
)

func main() {
	if err := run(); err != nil {
		exitError := &exec.ExitError{
			ProcessState: nil,
			Stderr:       nil,
		}
		if errors.As(err, &exitError) {
			// Swallow error message - it was printed via os.Stderr redirection.
			os.Exit(exitError.ExitCode())
		}
		if errString := err.Error(); errString != "" {
			_, _ = fmt.Fprintf(os.Stderr, "bufisk: %s\n", errString)
		}
		os.Exit(1)
	}
}

func run() error {
	ctx, cancel := withCancelInterruptSignal(context.Background())
	defer cancel()
	bufVersion, err := getBufVersion()
	if err != nil {
		return err
	}
	cacheDirPath := os.Getenv(cacheDirPathEnvKey)
	if cacheDirPath == "" {
		cacheDirPath, err = getDefaultCacheDirPath()
		if err != nil {
			return err
		}
		cacheDirPath = filepath.Join(cacheDirPath, "bufisk")
	}
	bufFilePath := filepath.Join(cacheDirPath, unameS, unameM, "releases", "buf", bufVersion, "bin", "buf")
	if _, err := os.Stat(bufFilePath); err != nil {
		if err := downloadBufToFilePath(ctx, bufVersion, bufFilePath); err != nil {
			return err
		}
		if _, err := fmt.Fprintf(os.Stderr, "bufisk: downloaded buf to %s\n\n", bufFilePath); err != nil {
			return err
		}
	}
	cmd := exec.CommandContext(ctx, bufFilePath, os.Args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Should do some check on the returned version format.
func getBufVersion() (string, error) {
	if useBufVersionEnvValue := os.Getenv(useBufVersionEnvKey); useBufVersionEnvValue != "" {
		return validateBufVersion(useBufVersionEnvValue, useBufVersionEnvKey)
	}
	pwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	curDirPath := pwd
	for {
		filePath := filepath.Join(curDirPath, bufVersionFileName)
		data, err := os.ReadFile(filePath)
		// Ignore all errors, not just fs.ErrNotExist - we don't want this program to fail
		// on bad permissions. We could choose to stop on the first bad permissions error.
		if err == nil {
			return validateBufVersion(strings.TrimSpace(string(data)), filePath)
		}
		// TODO: Not sure if works for Windows.
		if curDirPath == string(os.PathSeparator) {
			break
		}
		curDirPath = filepath.Dir(curDirPath)
	}
	return "", fmt.Errorf("%s not set and no %s file found", useBufVersionEnvKey, bufVersionFileName)
}

// We could import a SemVer library but this should be enough for now.
func validateBufVersion(bufVersion string, source string) (string, error) {
	split := strings.Split(bufVersion, ".")
	if len(split) != 3 {
		return "", newInvalidBufVersionError(bufVersion, source)
	}
	for _, s := range split {
		if _, err := strconv.Atoi(s); err != nil {
			return "", newInvalidBufVersionError(bufVersion, source)
		}
	}
	return bufVersion, nil
}

func downloadBufToFilePath(ctx context.Context, bufVersion string, bufFilePath string) (retErr error) {
	fileName := fmt.Sprintf("buf-%s-%s%s", unameS, unameM, executableSuffix)
	tempFilePath, err := downloadTempFile(ctx, getFileURL(bufVersion, fileName))
	if err != nil {
		return fmt.Errorf("could not download buf (are you sure %q is a valid release version?): %w", bufVersion, err)
	}
	defer func() {
		if err := os.Remove(tempFilePath); err != nil && retErr == nil {
			retErr = fmt.Errorf("failed to remove source file %q: %w", tempFilePath, err)
		}
	}()
	sha256TxtData, err := downloadData(ctx, getFileURL(bufVersion, "sha256.txt"))
	if err != nil {
		return err
	}
	sha256TxtMinisigData, err := downloadData(ctx, getFileURL(bufVersion, "sha256.txt.minisig"))
	if err != nil {
		return err
	}
	if err := verifySha256TxtData(sha256TxtData, sha256TxtMinisigData); err != nil {
		return err
	}
	sha256ExpectedHex, err := getSha256HexForTxtData(sha256TxtData, fileName)
	if err != nil {
		return err
	}
	sha256Hex, err := hashFile(tempFilePath)
	if err != nil {
		return fmt.Errorf("could not hash %s: %w", tempFilePath, err)
	}
	if sha256Hex != sha256ExpectedHex {
		return fmt.Errorf("sha256 mismatch for %s: expected %q got %q", fileName, sha256Hex, sha256ExpectedHex)
	}
	if err := copyFile(tempFilePath, bufFilePath); err != nil {
		return err
	}
	if err := os.Chmod(bufFilePath, 0700); err != nil {
		return err
	}
	return nil
}

func hashFile(filePath string) (hashStr string, retErr error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := file.Close(); err != nil {
			retErr = err
		}
	}()
	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func verifySha256TxtData(sha256TxtData []byte, sha256TxtMinisigData []byte) error {
	var publicKey minisign.PublicKey
	if err := publicKey.UnmarshalText([]byte(minisignPublicKey)); err != nil {
		return err
	}
	var signature minisign.Signature
	if err := signature.UnmarshalText(sha256TxtMinisigData); err != nil {
		return err
	}
	if signature.KeyID != publicKey.ID() {
		return fmt.Errorf("minisign key IDs for sha256.txt do not match:  ID (public key): %X ID (signature): %X", publicKey.ID(), signature.KeyID)
	}
	rawSignature, err := signature.MarshalText()
	if err != nil {
		return err
	}
	if !minisign.Verify(publicKey, sha256TxtData, rawSignature) {
		return errors.New("minisign signature verification of sha256.txt failed")
	}
	return nil
}

// Downloads to a temp file and returns the file path.
func downloadTempFile(ctx context.Context, url string) (string, error) {
	var tempFilePath string
	if err := download(
		ctx,
		url,
		func(reader io.Reader) (retErr error) {
			file, err := os.CreateTemp("", "bufisk*")
			if err != nil {
				return err
			}
			defer func() {
				if err := file.Close(); err != nil && retErr == nil {
					retErr = err
				}
			}()
			if _, err := io.Copy(file, reader); err != nil {
				return err
			}
			tempFilePath = file.Name()
			return nil
		},
	); err != nil {
		return "", err
	}
	return tempFilePath, nil
}

func downloadData(ctx context.Context, url string) ([]byte, error) {
	var data []byte
	if err := download(
		ctx,
		url,
		func(reader io.Reader) error {
			var err error
			data, err = io.ReadAll(reader)
			return err
		},
	); err != nil {
		return nil, err
	}
	return data, nil
}

func download(ctx context.Context, url string, processResponseBody func(io.Reader) error) (retErr error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	// Maybe don't use default client.
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return err
	}
	defer func() {
		if err := response.Body.Close(); err != nil && retErr == nil {
			retErr = err
		}
	}()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: HTTP Status %d", url, response.StatusCode)
	}
	return processResponseBody(response.Body)
}

func copyFile(fromFilePath string, toFilePath string) (retErr error) {
	if err := os.MkdirAll(filepath.Dir(toFilePath), 0700); err != nil {
		return err
	}
	inputFile, err := os.Open(fromFilePath)
	if err != nil {
		return err
	}
	defer func() {
		if err := inputFile.Close(); err != nil && retErr == nil {
			retErr = err
		}
	}()
	outputFile, err := os.Create(toFilePath)
	if err != nil {
		return err
	}
	defer func() {
		if err := outputFile.Close(); err != nil && retErr == nil {
			retErr = err
		}
	}()
	if _, err := io.Copy(outputFile, inputFile); err != nil {
		return err
	}
	return nil
}

func getFileURL(bufVersion string, fileName string) string {
	return fmt.Sprintf("https://github.com/bufbuild/buf/releases/download/v%s/%s", bufVersion, fileName)
}

func getSha256HexForTxtData(sha256TxtData []byte, fileName string) (string, error) {
	// Just assuming we actually have \n in our uploaded file.
	for _, line := range strings.Split(string(sha256TxtData), "\n") {
		if strings.HasSuffix(line, fileName) {
			split := strings.Split(line, "  ")
			if len(split) != 2 {
				return "", fmt.Errorf("invalid sha256.txt line: %q", line)
			}
			return split[0], nil
		}
	}
	return "", fmt.Errorf("could not find %q in sha256.txt:\n%s", fileName, string(sha256TxtData))
}

// withCancelInterruptSignal returns a context that is cancelled if interrupt signals are sent.
func withCancelInterruptSignal(ctx context.Context) (context.Context, context.CancelFunc) {
	interruptSignalC, closer := newInterruptSignalChannel()
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		<-interruptSignalC
		closer()
		cancel()
	}()
	return ctx, cancel
}

// newInterruptSignalChannel returns a new channel for interrupt signals.
//
// Call the returned function to cancel sending to this channel.
func newInterruptSignalChannel() (<-chan os.Signal, func()) {
	signalC := make(chan os.Signal, 1)
	signal.Notify(signalC, interruptSignals...)
	return signalC, func() {
		signal.Stop(signalC)
		close(signalC)
	}
}

func newInvalidBufVersionError(bufVersion string, source string) error {
	return fmt.Errorf(`invalid buf version from %s (must be in the form "MAJOR.MINOR.PATCH"): %q`, source, bufVersion)
}
