package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/joho/godotenv"
)

type OpenWRTLoginResponse struct {
	Expires int64  `json:"expires"`
	Timeout int64  `json:"timeout"`
	Token   string `json:"ubus_rpc_session"`
}

/*
{"success":true,"data":[{"txbytes":12332234942.0,"volte":false,"pinleft":3,"sc_band_av":"Single","rxbytes":156511412785.0,"wwan_gnss_conflict":false,"cell_info":[{"cellid":"76957227","mnc":"01","ue_state":3,"earfcn":1350,"pcid":17,"mcc":"222"}],"rscp":"N\/A","pinstate":"Inserted","provider":"TIM","oper":"I TIM","ecio":"N\/A","sinr":0,"state_id":1,"conntype":"4G (LTE)","state":"Connected","temperature":43,"version":"EG06ELAR04A04M4G","model":"EG06-E","busy_state":"Idle","imei":"868759037513647","active_sim":1,"imsi":"222014505262927","busy_state_id":1,"name":"Internal modem","pinstate_id":1,"iccid":"89390100002608548966","sim_count":1,"service_modes":{"4G":["lte_b1","lte_b3","lte_b5","lte_b7","lte_b8","lte_b20","lte_b28","lte_b32","lte_b38","lte_b40","lte_b41"],"3G":["wcdma_850","wcdma_900","wcdma_1800","wcdma_2100"]},"builtin":true,"cellid":"76957227","ipv6":true,"ca_signal":[],"baudrate":115200,"mode":0,"dynamic_mtu":true,"operators_scan":true,"band":"LTE B3","ntype":"LTE","volte_supported":true,"multi_apn":true,"id":"1-1","primary":true,"netstate":"Registered, home","rsrq":-12,"rsrp":-102,"signal":-62,"simstate_id":0,"lac":"4025","netstate_id":1,"pukleft":10,"is_busy":0,"simstate":"Inserted"}]}
*/
type OpenWRTModemStatus struct {
	Success bool `json:"success"`
	Data    []struct {
		TxBytes          float64 `json:"txbytes"`
		Volte            bool    `json:"volte"`
		PinLeft          int     `json:"pinleft"`
		ScBandAv         string  `json:"sc_band_av"`
		RxBytes          float64 `json:"rxbytes"`
		WwanGnssConflict bool    `json:"wwan_gnss_conflict"`

		CellInfo []struct {
			CellId  string `json:"cellid"`
			Mnc     string `json:"mnc"`
			UeState int    `json:"ue_state"`
			Earfcn  int    `json:"earfcn"`
			Pcid    int    `json:"pcid"`
			Mcc     string `json:"mcc"`
		} `json:"cell_info"`

		Rscp         string `json:"rscp"`
		PinState     string `json:"pinstate"`
		Provider     string `json:"provider"`
		Oper         string `json:"oper"`
		Ecio         string `json:"ecio"`
		Sinr         int    `json:"sinr"`
		StateId      int    `json:"state_id"`
		ConnType     string `json:"conntype"`
		State        string `json:"state"`
		Temperature  int    `json:"temperature"`
		Version      string `json:"version"`
		Model        string `json:"model"`
		BusyState    string `json:"busy_state"`
		Imei         string `json:"imei"`
		ActiveSim    int    `json:"active_sim"`
		Imsi         string `json:"imsi"`
		BusyStateId  int    `json:"busy_state_id"`
		Name         string `json:"name"`
		PinStateId   int    `json:"pinstate_id"`
		Iccid        string `json:"iccid"`
		SimCount     int    `json:"sim_count"`
		ServiceModes struct {
			Lte []string `json:"4G"`
		} `json:"service_modes"`
		Builtin bool   `json:"builtin"`
		CellId  string `json:"cellid"`

		Ipv6     bool     `json:"ipv6"`
		CaSignal []string `json:"ca_signal"`

		BaudRate       int    `json:"baudrate"`
		Mode           int    `json:"mode"`
		DynamicMtu     bool   `json:"dynamic_mtu"`
		OperatorsScan  bool   `json:"operators_scan"`
		Band           string `json:"band"`
		NType          string `json:"ntype"`
		VolteSupported bool   `json:"volte_supported"`
		MultiApn       bool   `json:"multi_apn"`
		Id             string `json:"id"`
		Primary        bool   `json:"primary"`
		NetState       string `json:"netstate"`
		Rsrq           int    `json:"rsrq"`
		Rsrp           int    `json:"rsrp"`
		Signal         int    `json:"signal"`
		SimStateId     int    `json:"simstate_id"`
		Lac            string `json:"lac"`
		NetStateId     int    `json:"netstate_id"`
		PukLeft        int    `json:"pukleft"`
		IsBusy         int    `json:"is_busy"`
		SimState       string `json:"simstate"`
	} `json:"data"`
}

var lastStatus = "ALIVE"

func heatbeatRoutine(mqttClient mqtt.Client, topic string, lastStatus string) {
	for {
		mqttClient.Publish(topic, 0, false, lastStatus)

		time.Sleep(time.Second)
	}
}

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	}

	var MQTT_BRORKER_URL = os.Getenv("MQTT_BRORKER_URL")
	var OPENWRT_URL = os.Getenv("OPENWRT_URL")
	var OPENWRT_USER = os.Getenv("OPENWRT_USER")
	var OPENWRT_PASSWORD = os.Getenv("OPENWRT_PASSWORD")
	var DELAY_SECONDS, _ = strconv.ParseInt(os.Getenv("DELAY_SECONDS"), 10, 64)
	var PUBLISH_TOPIC = os.Getenv("PUBLISH_TOPIC")
	var HEARTBEAT_TOPIC = os.Getenv("HEARTBEAT_TOPIC")

	opts := mqtt.NewClientOptions().AddBroker(MQTT_BRORKER_URL)
	opts.AutoReconnect = true
	opts.ConnectRetryInterval = 5 * time.Second
	mqttClient := mqtt.NewClient(opts)

	for {
		if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
			lastStatus = "ERROR"
			println(token.Error())
			time.Sleep(5 * time.Second)
			continue
		}
		defer mqttClient.Disconnect(250)
		break
	}

	go heatbeatRoutine(mqttClient, HEARTBEAT_TOPIC, lastStatus)

	for {
		time.Sleep(time.Duration(DELAY_SECONDS) * time.Second)

		loginReq, err := http.NewRequest("POST", OPENWRT_URL+"/api/login", nil)
		if err != nil {
			println(err)
			lastStatus = "ERROR"
			continue
		}

		loginReq.Header.Set("Content-Type", "application/json")
		loginReq.SetBasicAuth(OPENWRT_USER, OPENWRT_PASSWORD)

		client := &http.Client{}
		loginRes, err := client.Do(loginReq)
		if err != nil {
			println(err)
			lastStatus = "ERROR"
			continue
		}

		var loginResponseBody OpenWRTLoginResponse
		json.NewDecoder(loginRes.Body).Decode(&loginResponseBody)

		statusReq, err := http.NewRequest("GET", OPENWRT_URL+"/api/mobile/modems/status_full/", nil)
		if err != nil {
			println(err)
			lastStatus = "ERROR"
			continue
		}
		statusReq.Header.Set("Content-Type", "application/json")
		statusReq.Header.Set("Authorization", "Bearer "+loginResponseBody.Token)

		statusRes, err := client.Do(statusReq)
		if err != nil {
			println(err)
			lastStatus = "ERROR"
			continue
		}

		var modemStatus OpenWRTModemStatus
		json.NewDecoder(statusRes.Body).Decode(&modemStatus)

		mqttClient.Publish(PUBLISH_TOPIC+"/mobile_signal", 0, false, fmt.Sprintf("%d", modemStatus.Data[0].Signal))
		mqttClient.Publish(PUBLISH_TOPIC+"/mobile_rsrp", 0, false, fmt.Sprintf("%d", modemStatus.Data[0].Rsrp))
		mqttClient.Publish(PUBLISH_TOPIC+"/band", 0, false, modemStatus.Data[0].Band)
		mqttClient.Publish(PUBLISH_TOPIC+"/ntype", 0, false, modemStatus.Data[0].NType)

		statusRes.Body.Close()
		loginRes.Body.Close()

		lastStatus = "ALIVE"
	}
}
