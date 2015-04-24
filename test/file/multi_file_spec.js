var commons = require('../commons')
var frisby = require('frisby');
var FormData = require('form-data');

frisby.create('Multi file')
    //Create upload
    .post(commons.host+'/upload')
    .afterJSON(function(upload) {
        var upload_id = upload.id;

        files = ['Naab', 'Master', 'Is', 'B','odj','i']
        for (  i = 0 ; i < files.length; i++)
        {
            file_name = files[i];

            //Post 1 file
            form = new FormData();
            form.append('file', "Bodji c'est mon copain", {
                  contentType: 'text/plain',
                  filename: file_name
            });


            frisby.create('Post file')
                .post(
                    commons.host+'/upload/'+upload_id+'/file',
                    form,
                    {
                        json: false,
                        headers: {
                          'X-UploadToken' : upload.uploadToken,
                          'content-type': 'multipart/form-data; boundary=' + form.getBoundary(),
                          'content-length': form.getLengthSync()
                        }
                    }
                )
                .expectStatus(200)
                .expectJSONTypes('', {
                    'id': String,
                    'fileName': String
                })
                .expectJSON('',{
                    'fileName': file_name
                })
                .afterJSON(function(file) {
                    //Get the file
                    file_id = file.id;
                    file_name = file.fileName

                    frisby.create('Get file')
                        .get(commons.host+'/file/'+upload_id+'/'+file_id+'/'+file_name )
                        .expectStatus(200)
                        .after(function(err, res, body) {
                            expect("Bodji c'est mon copain").toEqual(body);
                         })
                .toss()
                })
            .toss()
        }
      })
 .toss()