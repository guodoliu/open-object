package common

import (
	"errors"
	"github.com/sevlyar/go-daemon"
	"k8s.io/klog/v2"
	"net"
	"os"
	"path/filepath"
	"strings"
)

func RunConnector() {
	cntxt := &daemon.Context{
		PidFileName: ConnectorPIDFilename,
		PidFilePerm: 0644,
		LogFileName: ConnectorLogFilename,
		LogFilePerm: 0640,
		WorkDir:     ConnectorWorkPath,
		Umask:       027,
	}

	d, err := cntxt.Reborn()
	if err != nil {
		klog.Fatalf("Unable to run connector: %s", err.Error())
	}
	if d != nil {
		return
	}
	defer func() {
		if err := cntxt.Release(); err != nil {
			klog.Errorf("Unable to release daemon: %v", err)
		}
	}()
	klog.Info("Fuse Connector Daemon Is Starting...")

	runFuseProxy()
}

func ConnectorRunInContainer(cmd string) (string, error) {
	c, err := net.Dial("unix", filepath.Join(HostDir, ConnectorSocketPath))
	if err != nil {
		klog.Infof("Fuse connector Dial error: %v", err)
		return err.Error(), err
	}
	defer c.Close()

	_, err = c.Write([]byte(cmd))
	if err != nil {
		return err.Error(), err
	}

	buf := make([]byte, 2048)
	n, err := c.Read(buf[:])
	if err != nil {
		return err.Error(), err
	}
	response := string(buf[0:n])
	if strings.HasPrefix(response, "Success") {
		respstr := response[8:]
		return respstr, nil
	}
	return response, errors.New("Fuse connector exec command error:" + response)
}

func runFuseProxy() {
	if IsDirExisting(ConnectorSocketPath) {
		if err := os.Remove(ConnectorSocketPath); err != nil {
			klog.Fatalf("fail to remove connector socket: %s", err.Error())
		}
	} else {
		pathDir := filepath.Dir(ConnectorSocketPath)
		if !IsDirExisting(pathDir) {
			if err := os.MkdirAll(pathDir, os.ModePerm); err != nil {
				klog.Fatalf("fail to mkdir: %s", err.Error())
			}
		}
	}

	klog.Infof("Socket path is ready: %s", ConnectorSocketPath)
	ln, err := net.Listen("unix", ConnectorSocketPath)
	if err != nil {
		klog.Fatalf("fail to listen: %s", err.Error())
	}
	klog.Info("Daemon Started ...")
	defer ln.Close()

	for {
		fd, err := ln.Accept()
		if err != nil {
			klog.Infof("Server Accept error: %s", err.Error())
			continue
		}
		go echoServer(fd)
	}
}

func echoServer(c net.Conn) {
	buf := make([]byte, 2048)
	nr, err := c.Read(buf)
	if err != nil {
		klog.Infof("Server Read error: %s", err.Error())
		return
	}

	cmd := string(buf[0:nr])
	klog.Infof("server receive csi command: %s", cmd)
	// run command
	if out, err := RunCommand(cmd); err != nil {
		reply := "Fail: " + cmd + ", error: " + err.Error()
		_, _ = c.Write([]byte(reply))
		klog.Info("server fail to run cmd: ", reply)
	} else {
		out = "Success:" + out
		_, _ = c.Write([]byte(out))
		klog.Info("server success to run cmd: ", out)
	}
}
