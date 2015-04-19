var commons = require('../commons')
var frisby = require('frisby');

frisby.create('Create simple upload')
    //Create upload
    .post(commons.host+'/upload')
    .expectStatus(200)
    .expectJSONTypes('', {
        'id': String,
        'oneShot': Boolean,
        'removable': Boolean
    })
    .expectJSON( '', {
        'oneShot': false,
        'removable': false
    })
    .afterJSON(function(upload) {
        expect(upload.uploadToken).toBeDefined();
        expect(upload.id).toBeDefined();
     })
    .toss()


frisby.create('Create complex upload')
    //Create upload
    .post(commons.host+'/upload', {
        removable:true,
        oneShot:true,
        comments:"Python Rox",
        fileNames:["snap.jpg"],
    },{ json:true })
    .expectStatus(200)
    .expectJSON( '', {
        'oneShot': true,
        'removable': true,
        'comments': "Python Rox"
    })
    .afterJSON(function(upload) {
        expect(upload.id).toBeDefined();
     })
    .toss()






