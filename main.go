package main

import (
    "flag"
    "fmt"
    "os"
    "os/signal"
    "time"

    //"github.com/nanoscopic/ios_video_stream/screencapture"
    "github.com/danielpaulus/quicktime_video_hack/screencapture"
    //cm "github.com/danielpaulus/quicktime_video_hack/screencapture/coremedia"
    //cm "github.com/nanoscopic/ios_video_stream/screencapture/coremedia"
    
    "go.nanomsg.org/mangos/v3"
	  "go.nanomsg.org/mangos/v3/protocol/push"
	  // register transports
	  _ "github.com/nanomsg/mangos/transport/all"
    
	  log "github.com/sirupsen/logrus"
)

func main() {
    var udid       = flag.String( "udid"     , ""                    , "Device UDID" )
    var devicesCmd = flag.Bool(   "devices"  , false                 , "List devices then exit" )
    var pullCmd    = flag.Bool(   "pull"   , false                   , "Pull video" )
    var pushSpec   = flag.String( "pushSpec" , "tcp://127.0.0.1:7878", "NanoMsg spec to push h264 nalus to" )
    var verbose    = flag.Bool(   "v"        , false                 , "Verbose Debugging" )
    flag.Parse()
    
    log.SetFormatter(&log.JSONFormatter{})

    if *verbose {
        log.Info("Set Debug mode")
        log.SetLevel(log.DebugLevel)
    }

    if *devicesCmd {
        devices(); return
    } else if *pullCmd {
        gopull( *pushSpec, *udid )
    } else {
        flag.Usage()
    }
}

func devices() {
    deviceList, err := screencapture.FindIosDevices()
    if err != nil { log.Errorf("Error finding iOS Devices - %s",err) }
    
    for _,device := range deviceList {
        fmt.Printf( "UDID:%s, Name:%s, VID=%s, PID=%s\n", device.SerialNumber, device.ProductName, device.VID, device.PID )
    }
}

func gopull( pushSpec string, udid string ) {
    pushSock := setup_nanomsg_sockets( pushSpec )    
    
    stopChannel := make( chan interface {} )
    stopChannel2 := make( chan interface {} )
    stopChannel3 := make( chan bool )
    waitForSigInt( stopChannel, stopChannel2, stopChannel3 )
    
    writer := NewNanoWriter( pushSock )
    
    attempt := 1
    for {
        success := startWithConsumer( writer, udid, stopChannel, stopChannel2 )
        if success {
            break
        }
        fmt.Printf("Attempt %i to start streaming\n", attempt)
        if attempt >= 4 {
            log.WithFields( log.Fields{
                "type": "stream_start_failed",
            } ).Fatal("Socket new error")
        }
        attempt++
        time.Sleep( time.Second * 1 )
    }
    
    <- stopChannel3
    writer.Stop()
}

func setup_nanomsg_sockets( pushSpec string ) ( pushSock mangos.Socket ) {
    var err error
    if pushSock, err = push.NewSocket(); err != nil {
        log.WithFields( log.Fields{
            "type": "err_socket_new",
            "spec": pushSpec,
            "err": err,
        } ).Fatal("Socket new error")
    }
    if err = pushSock.Dial(pushSpec); err != nil {
        log.WithFields( log.Fields{
            "type": "err_socket_connect",
            "spec": pushSpec,
            "err": err,
        } ).Fatal("Socket connect error")
    }
    
    return pushSock
}

func startWithConsumer( consumer screencapture.CmSampleBufConsumer, udid string, stopChannel chan interface{}, stopChannel2 chan interface{} )  ( bool ) {
    device, err := screencapture.FindIosDevice(udid)
    if err != nil { log.Errorf("no device found to activate - %s",err); return false }

    device, err = screencapture.EnableQTConfig(device)
    if err != nil { log.Errorf("Error enabling QT config - %s",err); return false }

    adapter := screencapture.UsbAdapter{}
 
    mp := screencapture.NewMessageProcessor( &adapter, stopChannel, consumer )

    err = adapter.StartReading( device, &mp, stopChannel2 )
    
    if err != nil { log.Errorf("failed connecting to usb - %s",err); return false }
    
    return true
}

func waitForSigInt( stopChannel chan interface{}, stopChannel2 chan interface{}, stopChannel3 chan bool ) {
    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt)
    go func() {
        for sig := range c {
            fmt.Printf("Got signal %s\n", sig)
            go func() { stopChannel3 <- true }()
            go func() {
                stopChannel2 <- true
                stopChannel2 <- true
            }()
            go func() {
                stopChannel <- true
                stopChannel <- true
            }()
            
        }
    }()
}
