# The postman Pechkin. Send file as an email attachment

```shell
Usage: pechkin [options] attachment_file

  -c, --config string   Config file name without extension (default /etc/pechkin, ./pechkin)
  -d, --debug           Debug flag (boolean)
  -?, --help            Help screen
  -m, --mailto string   Mail to address
  -s, --sleep int       Sleep time (secs)
  -t, --table string    Config section(table) name (default "general")
```

## Configuration parameters

| Name            | Description
| ----------------|-------------------------------------------------------------------------------
|mail_server      | server hostname or ip address (eg. mail.contoso.org)
|mail_server_port | port number (eg. 25,465)
|mail_server_ssl  | ssl support (yes/no)
|auth_user        | smtp auth username
|auth_pass        | smtp auth password
|skip_cert_verify | skip certificate verification (allow selfsigned) (yes/no)
|mail_from        | smtp mail from address
|mail_from_name   | message header From:
|mail_to          | smtp mail to address
|mail_to_name     | message header To:
|mail_to_cc       | carbon copy address
|mail_to_bcc      | black carbon copy address
|msg_subj         | message Subject:
|msg_text         | message body
|log_file         |  
|attach_file      | path to attachment (%s replaced by command line parameter )
|max_file_size    | max attachment size (bytes)
|copy_to_path     | copy attachment file to path
|match_name       | send attach only if name matches regexp
|skip_name        | do not send attach if name matches regexp
