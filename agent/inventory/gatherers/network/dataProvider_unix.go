// +build darwin freebsd linux netbsd openbsd

package network

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/aliyun/aliyun_assist_client/agent/log"

	"github.com/aliyun/aliyun_assist_client/agent/inventory/model"
)

// CollectNetworkData collects network information for linux
func collectNetworkData(config model.Config) (data []model.NetworkData) {

	//TODO: collect dhcp, dns server info from dhcp lease
	//TODO: collect gateway addresses (possibly from 'route -n')
	//TODO: collect subnetmask

	var interfaces []net.Interface
	var err error

	log.GetLogger().Debug("Detecting all network interfaces")

	interfaces, err = net.Interfaces()

	if err != nil {
		log.GetLogger().Debug("Unable to get network interface information")
		return
	}

	if interfaces != nil && len(interfaces) > NetworkConfigCountLimit {
		err = fmt.Errorf(NetworkConfigCountLimitExceeded+", got %d", len(interfaces))
		log.GetLogger().WithError(err).Error("collect network config failed")
		return []model.NetworkData{}
	}

	for _, i := range interfaces {
		var networkData model.NetworkData

		if i.Flags&net.FlagLoopback != 0 {
			log.GetLogger().Debug("Ignoring loopback interface")
			continue
		}

		networkData = setNetworkData(i)

		dataB, _ := json.Marshal(networkData)

		log.GetLogger().Debugf("Detected interface %v - %v", networkData.Name, string(dataB))
		data = append(data, networkData)
	}

	return
}

// setNetworkData sets network data using the given interface
func setNetworkData(networkInterface net.Interface) model.NetworkData {
	var addresses []net.Addr
	var err error

	var networkData = model.NetworkData{}

	networkData.Name = networkInterface.Name
	networkData.MacAddress = networkInterface.HardwareAddr.String()

	//getting addresses associated with network interface
	if addresses, err = networkInterface.Addrs(); err != nil {
		log.GetLogger().Debugf("Can't find address associated with %v", networkInterface.Name)
	} else {
		//TODO: current implementation is tied to inventory model where IPaddress is a string
		//if there are multiple ip addresses attached to an interface - we would overwrite the
		//ipaddresses. This behavior will be changed soon.
		for _, addr := range addresses {
			var ip net.IP

			switch v := addr.(type) {
			case *net.IPAddr:
				ip = v.IP
			case *net.IPNet:
				ip = v.IP
			}

			//To4 - return nil if address is not IPV4 address
			//we leverage this to determine if address is IPV4 or IPV6
			v4 := ip.To4()

			if len(v4) == 0 {
				networkData.IPV6 = ip.To16().String()
			} else {
				networkData.IPV4 = v4.String()
			}
		}
	}

	return networkData
}
