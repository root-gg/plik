
## yubigo

Yubigo is a Yubikey client API library that provides an easy way to integrate the Yubikey into any Go application.

## Installation

Installation is simple. Use go get:
`go get github.com/GeertJohan/yubigo`

## Usage

Make sure to import the library: `import "github.com/GeertJohan/yubigo"`

For use with the default Yubico servers, make sure you have an API key. [Request a key][getapikey].

**Basic OTP checking usage:**
```go

// create a new yubiAuth instance with id and key
yubiAuth, err := yubigo.NewYubiAuth("1234", "fdsaffqaf4vrc2q3cds=")
if err != nil {
	// probably an invalid key was given
	log.Fatalln(err)
}

// verify an OTP string
result, ok, err := yubiAuth.Verify("ccccccbetgjevivbklihljgtbenbfrefccveiglnjfbc")
if err != nil {
	log.Fatalln(err)
}

if ok {
	// succes!! The OTP is valid!
	log.Printf("Used query was: %s\n", result.GetRequestQuery()) // this query string includes the url of the api-server that responded first.
} else {
	// fail! The OTP is invalid or has been used before.
	log.Println("The given OTP is invalid!!!")
}
```


**Do not verify HTTPS certificate:**
```go
// Disable HTTPS cert verification. Use true to enable again.
yubiAuth.HttpsVerifyCertificate(false)
```


**HTTP instead of HTTPS:**
```go
// Disable HTTPS. Use true to enable again.
yubiAuth.UseHttps(false)
```


**Custom API server:**
```go
// Set a list of n servers, each server as host + path. 
// Do not prepend with protocol
yubiAuth.SetApiServerList("api0.server.com/api/verify", "api1.server.com/api/verify", "otherserver.com/api/verify")
```

## Licence

This project is licensed under a Simplified BSD license. Please read the [LICENSE file][license].


## Todo
 - Test files
 - More documentation
 - Getters/Setters for some options on the YubiAuth object.

## Protocol & Package documentation

This project is implementing a pure-Go Yubico OTP Validation Client and is following the [Yubico Validation Protocol Version 2.0][validationProtocolV20].

You will find "go doc"-like [package documentation at go.pkgdoc.org][pkgdoc].


 [license]: https://github.com/GeertJohan/yubigo/blob/master/LICENSE
 [getapikey]: https://upgrade.yubico.com/getapikey/
 [pkgdoc]: http://go.pkgdoc.org/github.com/GeertJohan/yubigo
 [validationProtocolV20]: http://code.google.com/p/yubikey-val-server-php/wiki/ValidationProtocolV20