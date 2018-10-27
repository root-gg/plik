== Plik SDK

This is the Golang Plik SDK.

It will allow you to connect to a plik instance from your Go projects, and make some file uploads.


First, you have to init a Plik Client :

	plik, err := sdk.NewClient("http://your.plik.url")
	if err != nil {
		return err
	}

Then you can create an upload 

    upload, err := plik.NewUpload()
	if err != nil {
		return err
	}


If you want custom options, such as "OneShot", "Removable", or set a custom TTL, you have another method :

    upload, err := c.NewUploadWithOptions(&sdk.UploadOptions{
        TTL: 30000,
        OneShot: true,
    })
	if err != nil {
		return err
	}


You are now ready to add files to the upload :

	file, err := upload.AddFileFromPath(upload, "test.txt")
	if err != nil {
		return err
	}

We do have another method to add file from an io.Reader, and a filename : 

    file, err := upload.AddFile(upload, reader, "test.txt")
	if err != nil {
		return err
	}


You can now start the upload of the file(s) :

	err = upload.Upload()
	if err != nil {
		return err
	}

That's it. Pretty simple isn't it ?
You can access the URL of the upload or the files :

	log.Printf("Upload URL : %s", upload.URL())

	for _, file := range upload.Files {
		log.Printf(" -> URL of %s : %s", file.Name, file.URL())
	}