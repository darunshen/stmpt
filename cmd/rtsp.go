/*
Copyright © 2020 author darunshen

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"

	"github.com/darunshen/stmpts/rtsp"
	"github.com/spf13/cobra"
)

// rtspCmd represents the rtsp command
var rtspCmd = &cobra.Command{
	Use:   "rtsp",
	Short: "start a rtsp media stream server",
	Long:  `start a rtsp media stream server.`,
	Run: func(cmd *cobra.Command, args []string) {
		address, err := cmd.Flags().GetString("address")
		if err != nil {
			fmt.Printf("get address error : %v\n", err)
		}
		readBuffer, err := cmd.Flags().GetInt("ReadBuffer")
		if err != nil {
			fmt.Printf("get ReadBuffer error : %v\n", err)
		}
		writeBuffer, err := cmd.Flags().GetInt("WriteBuffer")
		if err != nil {
			fmt.Printf("get WriteBuffer error : %v\n", err)
		}
		pushChannelBuffer, err := cmd.Flags().GetInt("PushChannelBuffer")
		if err != nil {
			fmt.Printf("get PushChannelBuffer error : %v\n", err)
		}
		pullChannelBuffer, err := cmd.Flags().GetInt("PullChannelBuffer")
		if err != nil {
			fmt.Printf("get PullChannelBuffer error : %v\n", err)
		}
		rtspServer := rtsp.Server{}
		err = rtspServer.Start(address, readBuffer, writeBuffer,
			pushChannelBuffer, pullChannelBuffer)
		if err != nil {
			fmt.Printf("rtspServer.Start error : %v\n", err)
		}
	},
}

func init() {
	serverCmd.AddCommand(rtspCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// rtspCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// rtspCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	rtspCmd.PersistentFlags().StringP("address", "a", "0.0.0.0:554",
		"the server's listen address")
	rtspCmd.PersistentFlags().IntP("ReadBuffer", "", 10485760,
		"ReadBuffer bio&tcp&udp read buffer size")
	rtspCmd.PersistentFlags().IntP("WriteBuffer", "", 10485760,
		"WriteBuffer bio&tcp&udp write buffer size")
	rtspCmd.PersistentFlags().IntP("PushChannelBuffer", "", 1,
		"pusher channel buffer size")
	rtspCmd.PersistentFlags().IntP("PullChannelBuffer", "", 1,
		"puller channel buffer size")
}
