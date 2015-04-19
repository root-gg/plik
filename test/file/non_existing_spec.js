var commons = require('../commons')
var frisby = require('frisby');
var FormData = require('form-data');

frisby.create('Get file')
    .get(commons.host+'/file/2134354/zgzhrthztrh/sgeh' )
    .after(function(err, res, body) {
        expect(res.request.href.match(/errcode=404/)).not.toBeNull()
     })
.toss()
