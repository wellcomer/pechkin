package main

// The postman Pechkin
// (c) 2018

import (
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	valid "github.com/asaskevich/govalidator"
	jww "github.com/spf13/jwalterweatherman"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	gomail "gopkg.in/gomail.v2"
)

type config struct {
	MailServer     string `mapstructure:"mail_server" valid:"host,required"`
	MailServerPort uint16 `mapstructure:"mail_server_port"`
	MailServerSSL  bool   `mapstructure:"mail_server_ssl"`
	AuthUser       string `mapstructure:"auth_user"`
	AuthPass       string `mapstructure:"auth_pass"`
	SkipCertVerify bool   `mapstructure:"skip_cert_verify"`
	MailFrom       string `mapstructure:"mail_from" valid:"email,required"`
	MailFromName   string `mapstructure:"mail_from_name"`
	MailTo         string `mapstructure:"mail_to" valid:"email,required"`
	MailToName     string `mapstructure:"mail_to_name"`
	MailToCC       string `mapstructure:"mail_to_cc" valid:"email"`
	MailToBCC      string `mapstructure:"mail_to_bcc" valid:"email"`
	MsgSubject     string `mapstructure:"msg_subj"`
	MsgText        string `mapstructure:"msg_text"`
	LogFile        string `mapstructure:"log_file"`
	AttachFile     string `mapstructure:"attach_file"`
	MaxFileSize    uint32 `mapstructure:"max_file_size"`
	CopyToPath     string `mapstructure:"copy_to_path"`
	MatchName      string `mapstructure:"match_name"`
	SkipName       string `mapstructure:"skip_name"`
}

// load config from .toml file

func loadConfig(configFile string, tableName string) *config {

	if configFile == "" {
		viper.AddConfigPath("/etc")    // look for config in the /etc directory
		viper.AddConfigPath(".")       // look for config in the working directory
		viper.SetConfigName("pechkin") // name of config file (without extension)
	} else {
		dir, file := path.Split(configFile)
		viper.AddConfigPath(dir)
		viper.SetConfigName(file)
	}

	err := viper.ReadInConfig() // find and read the config file
	if err != nil {             // handle errors reading the config file
		jww.FATAL.Fatalf("%s", err)
	}

	conf := config{}

	err = viper.UnmarshalKey("general", &conf)
	if err != nil {
		jww.FATAL.Fatalf("Unable to decode into struct %v", err)
	}
	err = viper.UnmarshalKey(tableName, &conf)
	if err != nil {
		jww.FATAL.Fatalf("Unable to decode into struct %v", err)
	}
	return &conf
}

// validate configuration parameters

func validateConfig(conf *config) {

	_, err := valid.ValidateStruct(conf)
	if err != nil {
		jww.FATAL.Fatalf("Failed to validate %s", err)
	}
}

func fileIsReadable(file string) bool {

	f, err := os.Open(file)
	if err != nil {
		jww.DEBUG.Printf("file error %s", err)
		return false
	}
	f.Close()
	return true
}

func fileIsSmaller(file string, maxSize int64) bool {

	fileInfo, err := os.Stat(file)
	if err != nil {
		return false
	}

	fileSize := fileInfo.Size()
	if fileSize < maxSize || maxSize == 0 {
		return true
	}

	jww.DEBUG.Printf("file size %d > file_max_size %d", fileSize, maxSize)
	return false
}

func copyFile(srcFile, dstFilePath string) (err error) {

	from, err := os.Open(srcFile)
	if err != nil {
		return
	}
	defer from.Close()

	_, srcFileName := path.Split(srcFile)
	dstFile := path.Join(dstFilePath, srcFileName)

	//srcFileInfo, _ := os.Stat(srcFile)
	//to, err := os.OpenFile(dstFile, os.O_RDWR|os.O_CREATE, srcFileInfo.Mode())
	to, err := os.OpenFile(dstFile, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return
	}
	defer to.Close()

	_, err = io.Copy(to, from)
	if err != nil {
		return
	}
	return
}

func main() {

	flagHelp := flag.BoolP("help", "?", false, "Help screen")
	flagDebug := flag.BoolP("debug", "d", false, "Debug flag (boolean)")
	flagConfigFile := flag.StringP("config", "c", "", "Config file name without extension (default /etc/pechkin, ./pechkin)")
	flagTableName := flag.StringP("table", "t", "general", "Config section(table) name")
	flagMailTo := flag.StringP("mailto", "m", "", "Mail to address")
	flagSleep := flag.IntP("sleep", "s", 0, "Sleep time (secs)")

	flag.Usage = func() {
		fmt.Printf("Usage: pechkin [options] attachment_file\n\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if *flagHelp {
		flag.Usage()
		os.Exit(0)
	}

	conf := loadConfig(*flagConfigFile, *flagTableName)

	if *flagMailTo != "" {
		conf.MailTo = *flagMailTo
	}

	validateConfig(conf)

	if conf.LogFile != "" {

		logfile, err := os.OpenFile(conf.LogFile, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0640)
		if err != nil {
			jww.FATAL.Fatalf("Error opening file: %s", err)
		}
		// set output for other levels
		jww.SetLogOutput(logfile)
		jww.SetPrefix(*flagTableName)

		// set output for INFO
		jww.INFO.SetFlags(log.Ldate | log.Ltime)
		jww.INFO.SetPrefix("[" + *flagTableName + "] ")
		jww.INFO.SetOutput(logfile)

	} else {
		jww.INFO.SetOutput(os.Stdout)
	}

	if *flagDebug {
		jww.DEBUG = jww.INFO
	}

	// dump args and configuration map
	jww.DEBUG.Printf("args %v", os.Args)
	jww.DEBUG.Printf("conf %+v", *conf)

	var attachName string

	// interpolate vars
	if len(flag.Args()) >= 1 {
		attachName = flag.Arg(0)
		if conf.AttachFile != "" {
			conf.AttachFile = fmt.Sprintf(conf.AttachFile, attachName)
		}
		jww.INFO.Printf("file %s", attachName)
	} else {
		jww.WARN.Printf("Empty argument!")
	}

	// process match_name, skip_name
	if attachName != "" {
		if conf.MatchName != "" {
			res, err := regexp.MatchString(conf.MatchName, attachName)
			if err != nil {
				jww.FATAL.Printf("error in regexp match_name %s", err)
			}
			if res != true {
				jww.DEBUG.Printf("name %s doesn't match_name %s", attachName, conf.MatchName)
				os.Exit(0)
			}
		}
		if conf.SkipName != "" {
			res, err := regexp.MatchString(conf.SkipName, attachName)
			if err != nil {
				jww.FATAL.Printf("error in regexp skip_name %s", err)
			}
			if res == true {
				jww.DEBUG.Printf("name %s matches skip_name %s", attachName, conf.SkipName)
				os.Exit(0)
			}
		}
	}

	// delay copy & attachment
	if *flagSleep > 0 {
		jww.INFO.Printf("sleep for %d seconds", *flagSleep)
		time.Sleep(time.Duration(*flagSleep) * time.Second)
	}

	// set default message subject
	if attachName != "" {
		if conf.MsgSubject == "" {
			conf.MsgSubject = "Attachment: %s"
		}
		if strings.ContainsAny(conf.MsgSubject, "%") {
			conf.MsgSubject = fmt.Sprintf(conf.MsgSubject, attachName)
		}
		if conf.MsgText != "" {
			conf.MsgText = fmt.Sprintf(conf.MsgText, attachName)
		}
	}

	// copy file to another path
	if conf.CopyToPath != "" && conf.AttachFile != "" {

		err := copyFile(conf.AttachFile, conf.CopyToPath)
		if err != nil {
			jww.INFO.Printf("copy error %s", err)
		} else {
			jww.INFO.Printf("copy %s to %s", conf.AttachFile, conf.CopyToPath)
		}
	}

	msg := gomail.NewMessage()
	msg.SetAddressHeader("From", conf.MailFrom, conf.MailFromName)
	msg.SetAddressHeader("To", conf.MailTo, conf.MailToName)

	if conf.MailToCC != "" {
		msg.SetAddressHeader("Cc", conf.MailToCC, "")
	}
	if conf.MailToBCC != "" {
		msg.SetAddressHeader("Bcc", conf.MailToBCC, "")
	}
	if conf.MsgSubject != "" {
		msg.SetHeader("Subject", conf.MsgSubject)
	}

	msg.SetBody("text/plain", conf.MsgText)

	if conf.AttachFile != "" {
		if fileIsReadable(conf.AttachFile) {
			if fileIsSmaller(conf.AttachFile, int64(conf.MaxFileSize)) {
				msg.Attach(conf.AttachFile)
				jww.INFO.Printf("file %s attached", conf.AttachFile)
			}
		}
	}

	server := gomail.Dialer{}
	server.Host = conf.MailServer
	server.Port = int(conf.MailServerPort)
	if server.Port == 0 {
		server.Port = 25
	}
	server.TLSConfig = &tls.Config{InsecureSkipVerify: conf.SkipCertVerify, ServerName: conf.MailServer}
	server.SSL = conf.MailServerSSL

	server.Username = conf.AuthUser
	server.Password = conf.AuthPass

	if err := server.DialAndSend(msg); err != nil {
		jww.FATAL.Fatalf("Failed to send message %s", err)
	}
	jww.INFO.Printf("mail ok %s", conf.MailTo)
}
