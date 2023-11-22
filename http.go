package main

import (
	"log"
	"net/http"
	"time"

	webrtc "github.com/deepch/vdk/format/webrtcv3"
	"github.com/gin-gonic/gin"
)

type JCodec struct {
	Type string
}

func serveHTTP() {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	router.OPTIONS("/"+Config.Server.AuthToken, CORSMiddleware())
	router.POST("/"+Config.Server.AuthToken, CORSMiddleware(), HTTPAPIServerStreamWebRTC)

	var err error
	if Config.Server.UseTLS {
		err = router.RunTLS(Config.Server.HTTPPort, Config.Server.TLSCert, Config.Server.TLSKey)
	} else {
		err = router.Run(Config.Server.HTTPPort)
	}
	if err != nil {
		log.Fatalln("Start HTTP Server error", err)
	}
}

func HTTPAPIServerStreamWebRTC(c *gin.Context) {
	uuid := "H264"

	if !Config.ext(uuid) {
		log.Println("Stream Not Found")
		return
	}
	Config.RunIFNotRun(uuid)
	codecs := Config.coGe(uuid)
	if codecs == nil {
		log.Println("Stream Codec Not Found")
		return
	}
	var AudioOnly bool
	if len(codecs) == 1 && codecs[0].Type().IsAudio() {
		AudioOnly = true
	}
	muxerWebRTC := webrtc.NewMuxer(webrtc.Options{ICEServers: Config.GetICEServers(), ICEUsername: Config.GetICEUsername(), ICECredential: Config.GetICECredential(), PortMin: Config.GetWebRTCPortMin(), PortMax: Config.GetWebRTCPortMax()})
	answer, err := muxerWebRTC.WriteHeader(codecs, c.PostForm("data"))
	if err != nil {
		log.Println("WriteHeader", err)
		return
	}
	_, err = c.Writer.Write([]byte(answer))
	if err != nil {
		log.Println("Write", err)
		return
	}
	go func() {
		cid, ch := Config.clAd(uuid)
		defer Config.clDe(uuid, cid)
		defer muxerWebRTC.Close()
		var videoStart bool
		noVideo := time.NewTimer(10 * time.Second)
		for {
			select {
			case <-noVideo.C:
				log.Println("noVideo")
				return
			case pck := <-ch:
				if pck.IsKeyFrame || AudioOnly {
					noVideo.Reset(10 * time.Second)
					videoStart = true
				}
				if !videoStart && !AudioOnly {
					continue
				}
				err = muxerWebRTC.WritePacket(pck)
				if err != nil {
					log.Println("WritePacket", err)
					return
				}
			}
		}
	}()
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		host := c.Request.Header.Get("Origin")
		if Config.Server.Host != host {
			c.Next()
			return
		}

		c.Header("Access-Control-Allow-Origin", host)
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		c.Header("Access-Control-Expose-Headers", "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers, Cache-Control, Content-Language, Content-Type")
		c.Header("Access-Control-Allow-Methods", "POST, OPTIONS")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

type Response struct {
	Tracks []string `json:"tracks"`
	Sdp64  string   `json:"sdp64"`
}

type ResponseError struct {
	Error string `json:"error"`
}
