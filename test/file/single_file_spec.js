var commons = require('../commons')
var frisby = require('frisby');
var FormData = require('form-data');

frisby.create('Single file')
    //Create upload
    .post(commons.host+'/upload')
    .afterJSON(function(upload) {
        //Post 1 file
        form = new FormData();
        form.append('file', "Bodji c'est mon copain", {
              contentType: 'text/plain',
              filename: 'test.bin'
        });

        var upload_id = upload.id;
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
                'fileName': 'test.bin'
            })
            .afterJSON(function(file) {
                //Get the file
                file_id = file.id;
                file_name = file.fileName;

                frisby.create('Get Single file')
                    .get(commons.host+'/file/'+upload_id+'/'+file_id+'/'+file_name )
                    .expectStatus(200)
                    .after(function(err, res, body) {
                        expect(body).toEqual("Bodji c'est mon copain");
                     })
            .toss()
            })
        .toss()
      })
 .toss()