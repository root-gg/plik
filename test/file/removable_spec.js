var commons = require('../commons')
var frisby = require('frisby');
var FormData = require('form-data');

frisby.create('Multi file')
    //Create upload
    .post(commons.host+'/upload',
        {
            removable: true
        },
        { json: true }
    )
    .afterJSON(function(upload) {
        var upload_id = upload.id;

        files = ['Naab', 'Master', 'Is', 'B','odj','i']
        for (  i = 0 ; i < files.length; i++)
        {
            file_name = files[i];

            //Post 1 file
            form = new FormData();
            form.append('file', "Bodji c'est plus mon copain", {
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

                .afterJSON(function(file) {
                    //Get the file
                    file_id = file.id;

                    frisby.create('Get file')
                        .delete(commons.host+'/upload/'+upload_id+'/file/'+file_id )
                        .expectStatus(200)
                        .afterJSON(function(rep) {
                            frisby.create('Get file')
                                .get(commons.host+'/file/'+upload_id+'/'+file_id+'/'+file_name )
                                .after(function(err, res, body) {
                                    expect(res.request.href.match(/errcode=404/)).not.toBeNull()
                                 })
                            .toss()
                        })
                    .toss()
                })
            .toss()
        }
      })
 .toss()