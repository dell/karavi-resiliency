//  Copyright © 2021-2025 Dell Inc. or its subsidiaries. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ssh

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"encoding/base64"
	"github.com/bramvdbogaerde/go-scp"
	"golang.org/x/crypto/ssh"
	"net"
)

// Client interface for the SSH command execution
type Client interface {
	Run(...string) error                       // Execute the array of commands and save the results
	HasError() bool                            // Returns true if there were any failures in the commands
	GetErrors() []string                       // Returns the error messages (if any exists for it)
	GetOutput() []string                       // Returns the output of the commands (if any exists for it)
	SendRequest(command string) error          // Sends command to the remote host, but does not wait for a reply
	Copy(srcFile, remoteFilepath string) error // Copy a local file to the remote file through the SSH session
}

// AccessInfo has information needed to make an SSH connection to a host
type AccessInfo struct {
	Hostname string // Hostname or IP to connect to
	Port     string // Port to use for SSH (default should be 22)
	Username string // Username cred for host
	Password string // Password cred for host
}

// CommandResult holds information about the result of the SSH commands run on the remote host
type CommandResult struct {
	commands  []string // commands that are to be run on the host
	output    []string // output for each command that was run (even in case of error)
	withError []bool   // Indicates if a command failed
	wasRun    []bool   // Indicates if a command was run
	err       error    // Error
}

// CommandExecution will hold information necessary for making SSH calls
// and then saving the results.
// CommandExecution implements the Client interface
type CommandExecution struct {
	AccessInfo *AccessInfo
	SSHWrapper ClientWrapper
	Timeout    time.Duration
	results    CommandResult
}

const (
	// DefaultTimeout for an operation
	DefaultTimeout = 1 * time.Hour
)

// Wrapper around the ssh.ClientConfig which is used for creating the
// underlying client to make SSH connections to a host
type Wrapper struct {
	SSHConfig  *ssh.ClientConfig
	sshClient  *ssh.Client
	sshSession *ssh.Session
	scpClient  scp.Client
}

// SessionWrapper interface for SSH session operations
//
//go:generate mockgen -destination=mocks/mock_session_wrapper.go -package=mocks podmon/test/ssh SessionWrapper
type SessionWrapper interface {
	CombinedOutput(string) ([]byte, error)
	Close() error
	SendRequest(name string, wantReply bool, payload []byte) (bool, error)
}

// ClientWrapper interface for creating an SSH session
//
//go:generate mockgen -destination=mocks/mock_client_wrapper.go -package=mocks podmon/test/ssh ClientWrapper
type ClientWrapper interface {
	GetSession(string) (SessionWrapper, error)
	Close() error
	SendRequest(name string, wantReply bool, payload []byte) (bool, error)
	Copy(ctx context.Context, srcFile os.File, remoteFilepath, permission string) error
}

// NewWrapper builds an ssh.ClientConfig and returns a Wrapper with it
func NewWrapper(accessInfo *AccessInfo) *Wrapper {
	config := &ssh.ClientConfig{
		User: accessInfo.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(accessInfo.Password),
		},
		HostKeyCallback: trustedHostKeyCallback(""),
	}
	wrapper := &Wrapper{SSHConfig: config}
	return wrapper
}

// create human-readable SSH-key strings
func keyString(k ssh.PublicKey) string {
	return k.Type() + " " + base64.StdEncoding.EncodeToString(k.Marshal())
}

func trustedHostKeyCallback(trustedKey string) ssh.HostKeyCallback {
	if trustedKey == "" {
		// accept any key, since the tests are run in a trusted non-prod environment
		return func(_ string, _ net.Addr, _ ssh.PublicKey) error {
			return nil
		}
	}
	return func(_ string, _ net.Addr, k ssh.PublicKey) error {
		ks := keyString(k)
		if trustedKey != ks {
			return fmt.Errorf("SSH-key verification: expected %q, but got %q", trustedKey, ks)
		}
		return nil
	}
}

// GetSession makes underlying call to crypto ssh library to create an SSH session
func (w *Wrapper) GetSession(hostAndPort string) (SessionWrapper, error) {
	client, err := ssh.Dial("tcp", hostAndPort, w.SSHConfig)
	if err == nil {
		w.sshClient = client
		w.sshSession, err = client.NewSession()
		if err != nil {
			return nil, err
		}
		w.scpClient, err = scp.NewClientBySSH(w.sshClient)
		return w.sshSession, err
	}
	return nil, fmt.Errorf("could not create a session")
}

// Close calls underlying crypto ssh library to clean up resources
func (w *Wrapper) Close() error {
	if w.sshClient != nil {
		if err := w.sshClient.Close(); err != nil {
			return err
		}
	}
	if w.sshSession != nil {
		if err := w.sshSession.Close(); err != nil {
			return err
		}
	}
	w.scpClient.Close()
	return nil
}

// GetClient returns the internal ssh.Client
func (w *Wrapper) GetClient() *ssh.Client {
	return w.sshClient
}

// SendRequest is a wrapper around the Session.SendRequest
func (w *Wrapper) SendRequest(name string, wantReply bool, payload []byte) (bool, error) {
	return w.sshSession.SendRequest(name, wantReply, payload)
}

// Copy is wrapper for the scpClient.CopyFromFile
func (w *Wrapper) Copy(ctx context.Context, srcFile os.File, remoteFilepath, permission string) error {
	return w.scpClient.CopyFromFile(ctx, srcFile, remoteFilepath, permission)
}

// Run will execute the commands using the AccessInfo to access the host.
// error returned by function is not related to the Run() result. For that
// result and any errors from the commands, use GetErrors() and GetOutput()
func (cmd *CommandExecution) Run(commands ...string) error {
	var err error
	timeout := time.After(DefaultTimeout)
	if cmd.Timeout != 0 {
		timeout = time.After(cmd.Timeout)
	}

	results := make(chan CommandResult, 1)
	go func() {
		r := cmd.execEach(commands)
		results <- r
	}()

	for {
		select {
		case r := <-results:
			cmd.results = r
			return r.err
		case <-timeout:
			msg := fmt.Sprintf("command '%s' on host %s timed out", strings.Join(commands, ","), cmd.AccessInfo.Hostname)
			err = fmt.Errorf("%s", msg)
			return err
		}
	}
}

// HasError returns true if there were any commands that failed
func (cmd *CommandExecution) HasError() bool {
	for _, e := range cmd.results.withError {
		if e {
			return true
		}
	}
	return false
}

// GetErrors returns an array of errors. Each entry in the
// array will contain the error of a command that failed.
func (cmd *CommandExecution) GetErrors() []string {
	list := make([]string, len(cmd.results.commands))
	if cmd.HasError() {
		idx := 0
		for index := range cmd.results.commands {
			if cmd.results.wasRun[index] && cmd.results.withError[index] {
				list[idx] = cmd.results.output[idx]
			}
		}
	}
	return list
}

// GetOutput returns the command output of each command if it was run
func (cmd *CommandExecution) GetOutput() []string {
	list := make([]string, len(cmd.results.commands))
	idx := 0
	for index, command := range cmd.results.commands {
		if cmd.results.wasRun[index] {
			list[idx] = cmd.results.output[idx]
		} else {
			list[idx] = fmt.Sprintf("Command not executed: %s", command)
		}
		idx = idx + 1
	}

	return list
}

// Copy will copy the file given by the 'srcFile' path to the remote host to the 'remoteFilePath' destination
func (cmd *CommandExecution) Copy(ctx context.Context, srcFile, remoteFilepath string) error {
	hostAndPort := fmt.Sprintf("%s:%s", cmd.AccessInfo.Hostname, cmd.AccessInfo.Port)
	_, err := cmd.SSHWrapper.GetSession(hostAndPort)
	if err != nil {
		return err
	}

	src, err := os.Open(srcFile)
	if err != nil {
		return err
	}

	err = cmd.SSHWrapper.Copy(ctx, *src, remoteFilepath, "0655")
	if err != nil {
		return err
	}
	cmd.cleanup()
	return nil
}

// SendRequest will send 'command' to the remote host without waiting for a reply
func (cmd *CommandExecution) SendRequest(command string) error {
	hostAndPort := fmt.Sprintf("%s:%s", cmd.AccessInfo.Hostname, cmd.AccessInfo.Port)
	client, err := cmd.SSHWrapper.GetSession(hostAndPort)
	if err != nil {
		return err
	}

	type execMsg struct {
		Command string
	}

	req := execMsg{
		Command: command,
	}

	_, err = client.SendRequest("exec", false, ssh.Marshal(req))
	cmd.cleanup()
	return err
}

// Private: execEach will run each command against the host and capture the output
func (cmd *CommandExecution) execEach(commands []string) CommandResult {
	// Connect to host
	hostAndPort := fmt.Sprintf("%s:%s", cmd.AccessInfo.Hostname, cmd.AccessInfo.Port)

	// Send the commands, then capture the output of each
	output := ""
	results := CommandResult{}
	results.commands = commands
	results.output = make([]string, len(commands))
	results.withError = make([]bool, len(commands))
	results.wasRun = make([]bool, len(commands))
	for index, command := range commands {
		client, err := cmd.SSHWrapper.GetSession(hostAndPort)
		if err != nil {
			return CommandResult{
				err: fmt.Errorf("could not connect to %s: %s", hostAndPort, err),
			}
		}
		output, err = cmd.exec(client, command)
		if err != nil {
			results.err = err
			results.withError[index] = true
		} else {
			results.withError[index] = false
		}
		results.output[index] = output
		results.wasRun[index] = true
		cmd.cleanup()
	}

	return results
}

// Private: cleanup will make underlying calls to clean up resources associated with SSH client and session
func (cmd *CommandExecution) cleanup() {
	_ = cmd.SSHWrapper.Close()
}

// Private: exec will run "command" on the host and wrap the []byte as a string
func (cmd *CommandExecution) exec(sess SessionWrapper, command string) (string, error) {
	// Execute the command and get output.
	output, err := sess.CombinedOutput(command)
	if err != nil {
		return string(output), err
	}

	return string(output), nil
}
