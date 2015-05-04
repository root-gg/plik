angular.module('dialog', ['ui.bootstrap']).
    factory('$dialog', function ($rootScope, $modal) {

        var module = {};

        // alert dialog
        module.alert = function (data) {
            if (!data) return false;
            var options = {
                backdrop: true,
                backdropClick: true,
                templateUrl: 'partials/alertDialogPartial.html',
                controller: 'AlertDialogController',
                resolve: {
                    args: function () {
                        return {
                            data: angular.copy(data)
                        }
                    }
                }
            };
            module.openDialog(options);
        };

        // generic dialog
        $rootScope.dialogs = [];
        module.openDialog = function (options) {
            if (!options) return false;

            $.each($rootScope.dialogs, function (i, dialog) {
                dialog.close();
            });
            $rootScope.dialogs = [];

            $modal.open(options);
        };

        return module;
    });

angular.module('api', ['angularFileUpload']).
    factory('$api', function ($http, $q, $upload) {
        var api = { base : '' };

        api.call = function(url,method,params){
            var promise = $q.defer();
            var data = params;
            $http({
                url: url,
                method: method,
                data: data
            })
            .success(function (data) {
                promise.resolve(data);
            })
            .error(function (data, code) {
                promise.reject({ status: code, message: data.message });
            });
            return promise.promise;
        }

        api.upload = function (url, file, params, progress_cb, basicAuth, uploadToken) {
            var promise = $q.defer();
            var headers = {};
            if (basicAuth) headers['Authorization'] = "Basic " + basicAuth;
            if (uploadToken) headers['X-UploadToken'] = uploadToken;
            $upload
                .upload({
                    url: url,
                    method: 'POST',
                    data: params,
                    fileName: file.uploadName,
                    file: file,
                    headers : headers
                })
                .progress(progress_cb)
                .success(function (data) {
                    promise.resolve(data);
                })
                .error(function (data, code) {
                    promise.reject({ status: code, message: data.message });
                });

            return promise.promise;
        };

        api.createUpload = function (params, names) {
            params.fileNames = names;
            var url = api.base + '/upload';
            return api.call(url,'POST',params);
        };

        api.getUpload = function (uploadId) {
            var url = api.base + '/upload/' + uploadId;
            return api.call(url,'GET',{});
        };

        api.getFile = function (uploadId, fileId) {
            var url = api.base + '/upload/' + uploadId + '/file/' + fileId;
            return api.call(url,'GET',{});
        };

        api.uploadFile = function (uploadId, file, progres_cb, basicAuth, uploadToken) {
        var url = api.base + '/upload/' + uploadId + '/file';
            return api.upload(url, file, {foo:"bar"}, progres_cb, basicAuth, uploadToken);
        };


        api.removeFile = function (uploadId, fileId) {
            var url = api.base + '/upload/' + uploadId + '/file/' + fileId;
            return api.call(url,'DELETE',{});
        };

        return api;
    });

angular.module('autoroot', ['ngRoute', 'api', 'dialog','contenteditable','ngClipboard','ngSanitize', 'btford.markdown'])
	.config(function($routeProvider) {
		$routeProvider
			.when('/', { controller:UploadCtrl, templateUrl:'partials/main.html', reloadOnSearch: false})
            .when('/clients', { controller:ClientListCtrl, templateUrl:'partials/clients.html'})
			.otherwise({ redirectTo: '/' });
    })
    .config(['$httpProvider', function($httpProvider) {
        $httpProvider.defaults.headers.common['X-ClientApp'] = 'web_client';
    }])
    .config(['ngClipProvider', function(ngClipProvider) {
        ngClipProvider.setPath("bower_components/zeroclipboard/dist/ZeroClipboard.swf");
    }]);


function UploadCtrl($scope, $dialog, $route, $location, $api) {
    $scope.sortField = 'metadata.fileName';
    $scope.sortOrder = false;

    $scope.upload = {};
    $scope.files = [];
    $scope.ttl = { value : 30, unit : "day", units : ["min","hour","day"] };
    $scope.yubikey = false;
    $scope.password = false;

    $scope.init = function(){
        // Display error from download redirect
        var err = $location.search().err;
        if ( ! _.isUndefined(err) ) {
            if ( err == "Invalid yubikey token" && $location.search().uri ) {
                $scope.load($location.search().uri.substr(6,16))
                $scope.downloadWithYubikey(location.origin + $location.search().uri)
            } else {
                var code = $location.search().errcode;
                $dialog.alert({ status: code, message: err });
            }
        } else {
            $scope.load($location.search().id)
        }
    };

    // Load upload from id
    $scope.load = function(id) {
        if(!id) return
        $scope.upload.id = id;
        $api.getUpload($scope.upload.id)
            .then(function(metadatas){
                console.log("metadatas", metadatas);
                _.extend($scope.upload,metadatas);
                $scope.files = _.map($scope.upload.files,function(metadata){
                    return { metadata : metadata };
                });
            })
            .then(null,function(error){
                $dialog.alert(error);
            });
    }

    // Add file to the upload list
    $scope.onFileSelect = function (files) {
        _.each(files, function (file) {
            // remove already added files
            var names = _.pluck($scope.files, 'name');
            if (!_.contains(names, file.name)) {
                file.uploadName = file.name;
                $scope.files.push(file);
            }
        });
    };

    // Remove file from the upload list
    $scope.removeFile = function (file) {
        $scope.files = _.reject($scope.files, function (f) {
            return f.name == file.name;
        });
    };

    // Create a new upload
    $scope.newUpload = function () {
        if (!$scope.files.length) return;
        console.log("new upload", $scope.upload);
        $scope.upload.ttl = $scope.getTTL()
        if($scope.password && ! ($scope.upload.login && $scope.upload.password)) {
            $scope.getPassword();
            return;
        }
        if($scope.yubikey && ! $scope.upload.yubikey) {
            $scope.getYubikey();
            return;
        }
        $api.createUpload($scope.upload, _.pluck($scope.files, 'name'))
            .then(function (upload) {
                console.log("metadatas", upload);
                $scope.upload = upload;
                $location.search('id', $scope.upload.id);
                $scope.uploadFiles();
            })
            .then(null, function (error) {
                $dialog.alert(error);
            });
    };

    // Upload every files
    $scope.uploadFiles = function () {
        if (!$scope.upload.id) return;
        _.each($scope.files, function (file) {
            if (file.metadata) return;
            var cb = function (event) {
                $scope.progress(file, event)
            };
            $api.uploadFile($scope.upload.id, file, cb, $scope.basicAuth, $scope.upload.uploadToken)
                .then(function (result) {
                    $scope.success(file, result);
                })
                .then(null, function (error) {
                    $scope.error(file, error);
                });
        });
    };

    // Remove a file from the servers
    $scope.delete = function(file) {
        $api.removeFile($scope.upload.id,file.metadata.id)
            .then(function (result){
                $scope.files = _.reject($scope.files, function (f) {
                    return f.metadata.fileName == file.metadata.fileName;
                });
                if (!$scope.files.length){
                    $location.search('id',null);
                    $route.reload();
                }
            })
            .then(function (error){
                $dialog.alert(error);
            });
    };

    $scope.progress = function (file, event) {
        file.progress = parseInt(100.0 * event.loaded / event.total);
    };

    $scope.success = function (file, result) {
        file.metadata = result;
    };

    $scope.error = function (file, error) {
        $dialog.alert(error);
    };

    $scope.humanReadableSize = function(size){
        if(_.isUndefined(size)) return;
        return filesize(size, { base : 2 });
    };

    $scope.getFileUrl = function(file,dl) {
        if(!file || !file.metadata) return;

        var url = location.origin + '/file/' + $scope.upload.id + '/' + file.metadata.id + '/' + file.metadata.fileName
        if(dl) {
            url += "?dl=1";
        }

        return url
    };

    $scope.getPassword = function() {
        var opts = {
            backdrop: true,
            backdropClick: true,
            templateUrl: 'partials/password.html',
            controller: 'PasswordController',
            resolve: {
                args: function () {
                    return {
                        callback: $scope.setCredentials
                    }
                }
            }
        };

        $dialog.openDialog(opts);
    };

    $scope.setCredentials = function(login,password) {
        $scope.upload.login = login;
        $scope.upload.password = password;
        $scope.basicAuth = btoa(login+":"+password);
        $scope.newUpload();
    };

    $scope.getYubikey = function() {
        var opts = {
            backdrop: true,
            backdropClick: true,
            templateUrl: 'partials/yubikey.html',
            controller: 'YubikeyController',
            resolve: {
                args: function () {
                    return {
                        callback: $scope.setYubikey
                    }
                }
            }
        };

        $dialog.openDialog(opts);
    };

    $scope.setYubikey = function(otp) {
        $scope.upload.yubikey = otp;
        $scope.newUpload();
    };

    $scope.downloadWithYubikey = function(url) {
        var opts = {
            backdrop: true,
            backdropClick: true,
            templateUrl: 'partials/yubikey.html',
            controller: 'YubikeyController',
            resolve: {
                args: function () {
                    return {
                        callback: function(token){
                            window.location.replace(url + '/yubikey/' + token);
                        }
                    }
                }
            }
        };

        $dialog.openDialog(opts);
    };

    $scope.switchTimeUnit = function() {
        var index = (_.indexOf($scope.ttl.units, $scope.ttl.unit) + 1) % $scope.ttl.units.length;
        $scope.ttl.unit = $scope.ttl.units[index];
    }

    $scope.getTTL = function() {
        var ttl = $scope.ttl.value
        if ( ttl > 0) {
            if ($scope.ttl.unit == "min") {
                ttl = ttl * 60;
            } else if ($scope.ttl.unit == "hour") {
                ttl = ttl * 3600;
            } else if ($scope.ttl.unit == "day") {
                ttl = ttl * 86400;
            }
        }
        return ttl;
    }

    $scope.init();
}

function ClientListCtrl($scope, $location) {
    $scope.clients = []
    
    $scope.addClient = function(name,arch,exe) {
        var binary = exe ? "plik.exe" : "plik"
        $scope.clients.push({name : name, url : location.origin + "/clients/" + arch + "/" + binary });
    }
    
    $scope.addClient("Linux 64bit","linux-amd64");
    $scope.addClient("Linux 32bit","linux-386");
    $scope.addClient("Linux ARM","linux-arm");
    $scope.addClient("MacOS 64bit","darwin-amd64");
    $scope.addClient("MacOS 32bit","darwin-386");
    $scope.addClient("Freebsd 64bit","freebsd-amd64");
    $scope.addClient("Freebsd 32bit","freebsd-386");
    $scope.addClient("Freebsd ARM","freebsd-arm");
    $scope.addClient("Openbsd 64bit","openbsd-amd64");
    $scope.addClient("Openbsd 32bit","openbsd-386");
    $scope.addClient("Windows 64bit","windows-amd64",true);
    $scope.addClient("Windows 32bit","windows-386",true);
}

function AlertDialogController($rootScope, $scope, $modalInstance, args) {
    $rootScope.dialogs.push($scope);

    $scope.title = 'Success !';
    if (args.data.status != 100) $scope.title = 'Oops !';

    $scope.data = args.data;

    $scope.close = function (result) {
        $rootScope.dialogs = _.without($rootScope.dialogs, $scope);
        $modalInstance.close(result);
        if(args.callback) {
            args.callback(result);
        }
    };
}

function PasswordController($rootScope, $scope, $modalInstance, args) {
    $rootScope.dialogs.push($scope);

    // Ugly but it works
    setTimeout(function () {
        $("#login").focus();
    }, 100);

    $scope.title = 'Please fill credentials !';
    $scope.login = "plik";
    $scope.password = "";

    $scope.check = function(login,password){
        if(login.length > 0 && password.length > 0){
            $scope.close(login,password);
        }
    };

    $scope.close = function (login,password) {
        if(!(login.length > 0 && password.length > 0)){
            return;
        }
        $scope.dismiss()
        if(args.callback) {
            args.callback(login,password);
        }
    };

    $scope.dismiss = function () {
        $rootScope.dialogs = _.without($rootScope.dialogs, $scope);
        $modalInstance.close();
    }
}

function YubikeyController($rootScope, $scope, $modalInstance, args) {
    $rootScope.dialogs.push($scope);

    // Ugly but it works
    setTimeout(function () {
        $("#yubikey").focus();
    }, 100);

    $scope.title = 'Please fill in a Yubikey OTP !';
    $scope.token = "";

    $scope.check = function(token){
        if(token.length == 44){
            $scope.close(token);
        }
    };

    $scope.close = function (result) {
        $scope.dismiss()
        if(args.callback) {
            args.callback(result);
        }
    };

    $scope.dismiss = function () {
        $rootScope.dialogs = _.without($rootScope.dialogs, $scope);
        $modalInstance.close();
    }
}