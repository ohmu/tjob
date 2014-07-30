/*
Package sshcmd - SSH command utility functions

Copyright (c) 2014 Ohmu Ltd.
Licensed under the Apache License, Version 2.0 (see LICENSE)
*/
package sshcmd

import (
	"code.google.com/p/go.crypto/ssh"
	"errors"
	"io/ioutil"
	"os"
	"path"
	"strconv"
)

type SSHPort int

func (s *SSHPort) String() string {
	return strconv.Itoa(int(*s))
}

type SSHNode struct {
	Host string
	Port SSHPort
	User string
	Key  string
}

func parseKey(file string) (ssh.Signer, error) {
	privateBytes, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, errors.New("failed to load private key")
	}
	private, err := ssh.ParsePrivateKey(privateBytes)
	if err != nil {
		return nil, errors.New("failed to parse private key")
	}
	return private, nil
}

func command(session *ssh.Session, cmd string) (string, error) {
	output, err := session.CombinedOutput(cmd)
	return string(output), err
}

func close(session *ssh.Session) {
	session.SendRequest("close", false, nil)
}

func (node *SSHNode) connect() (*ssh.Client, error) {
	key := node.Key
	if key == "" {
		key = "id_rsa"
	}
	if !path.IsAbs(key) {
		key = path.Join(os.Getenv("HOME"), ".ssh", key)
	}
	pkey, err := parseKey(key)
	if err != nil {
		return nil, err
	}
	user := node.User
	if user == "" {
		user = os.Getenv("LOGNAME")
	}
	if user == "" {
		return nil, errors.New("ssh login user not defined")
	}
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{ssh.PublicKeys(pkey)},
	}
	port := node.Port
	if port == 0 {
		port = 22
	}
	client, err := ssh.Dial("tcp", node.Host+":"+port.String(), config)
	if err != nil {
		return nil, errors.New("SSH connect failed: " + err.Error())
	}
	return client, nil
}

func (node *SSHNode) Execute(cmd string) (output string, err error) {
	client, err := node.connect()
	if err != nil {
		return "", err
	}
	session, err := client.NewSession()
	if err != nil {
		return "", errors.New(
			"Failed to create SSH session: " + err.Error())
	}
	defer close(session)
	defer session.Close()

	return command(session, cmd)
}
