var commons = require('../commons')
var frisby = require('frisby');
var FormData = require('form-data');


files = {
    '../../coucou': 'coucou',
    'ƒÂƒêÂ∑∆ﬂÎŸ‚ÍªËÛªŸ∫‚Ê' : 'ƒÂƒêÂ∑∆ﬂÎŸ‚ÍªËÛªŸ∫‚Ê'
}


frisby.create('file wierd name')
    .post(commons.host+'/upload')
    .afterJSON(function(upload) {
        var upload_id = upload.id;

        for(var file_name in files)
        {

            form = new FormData();
            form.append('file', "Bodji c'est mon copain", {
                  contentType: 'text/plain',
                  filename: file_name
            });


            frisby.create('Post file' + file_name)
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
                    'fileName': files[file_name]
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