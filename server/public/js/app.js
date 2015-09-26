/* The MIT License (MIT)

 Copyright (c) <2015>
 - Mathieu Bodjikian <mathieu@bodjikian.fr>
 - Charles-Antoine Mathieu <skatkatt@root.gg>

 Permission is hereby granted, free of charge, to any person obtaining a copy
 of this software and associated documentation files (the "Software"), to deal
 in the Software without restriction, including without limitation the rights
 to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 copies of the Software, and to permit persons to whom the Software is
 furnished to do so, subject to the following conditions:

 The above copyright notice and this permission notice shall be included in
 all copies or substantial portions of the Software.

 THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 THE SOFTWARE. */

// Editable file name directive
angular.module('contentEditable', []).
    directive('contenteditable', [ function() {
        return {
            restrict: 'A',          // only activate on element attribute
            require: '?ngModel',    // get a hold of NgModelController
            scope: {
                invalidClass: '@',  // Bind invalid-class attr evaluated expr
                validator: '&'      // Bind parent scope value
            },
            link: function(scope, element, attrs, ngModel) {
                if (!ngModel) return; // do nothing if no ng-model
                scope.validator = scope.validator(); // ???

                // Update view from model
                ngModel.$render = function() {
                    var string = ngModel.$viewValue;
                    validate(string);
                    element.text(string);
                };

                // Update model from view
                function update() {
                    var string = element.text();
                    validate(string);
                    ngModel.$setViewValue(string);
                }

                // Validate input and update css class
                function validate(string) {
                    if (scope.validator){
                        if(scope.validator(string)){
                            element.removeClass(scope.invalidClass);
                        } else {
                            element.addClass(scope.invalidClass);
                        }
                    }
                }

                // Listen for change events to enable binding
                element.on('blur keyup change', function() {
                    scope.$evalAsync(update);
                });
            }
        };
    }]);

// Modal dialog service
angular.module('dialog', ['ui.bootstrap']).
    factory('$dialog', function ($rootScope, $modal) {

        $rootScope.dialogs = [];

        // Register dialog
        $rootScope.registerDialog = function($dialog){
            $rootScope.dialogs.push($dialog);
        };

        // Dismiss dialog
        $rootScope.dismissDialog = function($dialog) {
            $rootScope.dialogs = _.without($rootScope.dialogs, $dialog);
        };

        var module = {};

        // alert dialog
        module.alert = function (data) {
            if (!data) return false;
            var options = {
                backdrop: true,
                backdropClick: true,
                templateUrl: 'partials/alert.html',
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

// API Service
angular.module('api', ['ngFileUpload']).
    factory('$api', function ($http, $q, Upload) {
        var api = {base: ''};

        // Make the actual HTTP call and return a promise
        api.call = function (url, method, params, uploadToken) {
            var promise = $q.defer();
            var headers = {};
            if (uploadToken) headers['X-UploadToken'] = uploadToken;
            $http({
                url: url,
                method: method,
                data: params,
                headers: headers
            })
                .success(function (data) {
                    promise.resolve(data);
                })
                .error(function (data, code) {
                    // Format HTTP error return for the dialog service
                    promise.reject({status: code, message: data.message});
                });
            return promise.promise;
        };

        // Make the actual HTTP call to upload a file and return a promise
        api.upload = function (url, file, params, progress_cb, basicAuth, uploadToken) {
            var promise = $q.defer();
            var headers = {};
            if (basicAuth) headers['Authorization'] = "Basic " + basicAuth;
            if (uploadToken) headers['X-UploadToken'] = uploadToken;
            Upload
                .upload({
                    url: url,
                    method: 'POST',
                    data: params,
                    fileName: file.metadata.fileName,
                    file: file,
                    headers: headers
                })
                .progress(progress_cb)
                .success(function (data) {
                    promise.resolve(data);
                })
                .error(function (data, code) {
                    // Format HTTP error return for the dialog service
                    promise.reject({status: code, message: data.message});
                });

            return promise.promise;
        };

        // Get upload metadata
        api.getUpload = function (uploadId) {
            var url = api.base + '/upload/' + uploadId;
            return api.call(url, 'GET', {});
        };

        // Create an upload with current settings
        api.createUpload = function (upload) {
            var url = api.base + '/upload';
            return api.call(url, 'POST', upload);
        };

        // Upload a file
        api.uploadFile = function (upload, file, progres_cb, basicAuth) {
            var mode = upload.stream ? "stream" : "file";
            var url = api.base + '/' + mode + '/' + upload.id + '/' + file.metadata.id + '/' + file.metadata.fileName;
            return api.upload(url, file, null, progres_cb, basicAuth, upload.uploadToken);
        };

        // Remove a file
        api.removeFile = function (upload, file) {
            var mode = upload.stream ? "stream" : "file";
            var url = api.base + '/' + mode + '/' + upload.id + '/' + file.metadata.id + '/' + file.metadata.fileName;
            return api.call(url, 'DELETE', {}, upload.uploadToken);
        };

        // Get server version
        api.getVersion = function() {
            var url = api.base + '/version';
            return api.call(url, 'GET', {});
        };

        // Get server config
        api.getConfig = function() {
            var url = api.base + '/config';
            return api.call(url, 'GET', {});
        };

        return api;
    });

// Plik app bootstrap and global configuration
angular.module('plik', ['ngRoute', 'api', 'dialog', 'contentEditable', 'btford.markdown'])
    .config(function ($routeProvider) {
        $routeProvider
            .when('/', {controller: MainCtrl, templateUrl: 'partials/main.html', reloadOnSearch: false})
            .when('/clients', {controller: ClientListCtrl, templateUrl: 'partials/clients.html'})
            .otherwise({redirectTo: '/'});
    })
    .config(['$httpProvider', function ($httpProvider) {
        $httpProvider.defaults.headers.common['X-ClientApp'] = 'web_client';
    }])
    .filter('collapseClass', function () {
        return function (opened) {
            if (opened) return "fa fa-caret-down";
            return "fa fa-caret-right";
        }
    });

// Main controller
function MainCtrl($scope, $dialog, $route, $location, $api) {
    $scope.sortField = 'metadata.fileName';
    $scope.sortOrder = false;

    $scope.upload = {};
    $scope.files = [];
    $scope.yubikey = false;
    $scope.password = false;

    // Get server config
    $api.getConfig()
        .then(function (config) {
            $scope.config = config;
            $scope.setDefaultTTL();
        })
        .then(null, function (error) {
            $dialog.alert(error);
        });

    // File name checks
    var fileNameMaxLength = 1024;
    var invalidCharList = ['/','#','?','%','"'];
    $scope.fileNameValidator = function(fileName) {
        if(_.isUndefined(fileName)) return false;
        if(fileName.length == 0 || fileName.length > fileNameMaxLength) return false;
        return _.every(invalidCharList, function(char){
            return fileName.indexOf(char) == -1;
        });
    };

    // Initialize main controller
    $scope.init = function () {
        // Display error from redirect if any
        var err = $location.search().err;
        if (!_.isUndefined(err)) {
            if (err == "Invalid yubikey token" && $location.search().uri) {
                var uri = $location.search().uri.split("/");
                $scope.load(uri[2]);
                $scope.downloadWithYubikey(location.origin + "/file/" + uri[2] + "/" + uri[3] + "/" + uri[4]);
            } else {
                var code = $location.search().errcode;
                $dialog.alert({status: code, message: err});
            }
        } else {
            // Load current upload id
            $scope.load($location.search().id);
            $scope.upload.uploadToken = $location.search().uploadToken;
        }
    };

    // Load upload from id
    $scope.load = function (id) {
        if (!id) return;
        $scope.upload.id = id;
        $api.getUpload($scope.upload.id)
            .then(function (upload) {
                _.extend($scope.upload, upload);
                $scope.files = _.map($scope.upload.files, function (file) {
                    return {metadata: file};
                });
            })
            .then(null, function (error) {
                $dialog.alert(error);
            });
    };

    // Reference is needed to match files ids
    var reference = -1;
    var nextRef = function () {
        reference++;
        return reference.toString();
    };

    // Detect shitty Apple devices
    $scope.isAppleShit = function(){
        return navigator.userAgent.match(/iPhone/i)
            || navigator.userAgent.match(/iPad/i)
            || navigator.userAgent.match(/iPod/i);
    };

    // Add a file to the upload list
    $scope.onFileSelect = function (files) {
        _.each(files, function (file) {
            // Check file size
            if($scope.config.maxFileSize && file.size > $scope.config.maxFileSize){
                $dialog.alert({
                    status: 0,
                    message: "File is too big : " + $scope.humanReadableSize(file.size),
                    value: "Maximum allowed size is : " + $scope.humanReadableSize($scope.config.maxFileSize)
                });
                return;
            }

            // Already added file names
            var names = _.pluck($scope.files, 'name');

            // iPhone/iPad/iPod fix
            // Apple mobile devices does not populate file name
            // well and tends to use something like image.jpg
            // every time a new image is selected.
            // If this appends an increment is added in the middle of
            // the filename ( image.1.jpg )
            // As a result of this the same file can be uploaded twice.
            if ($scope.isAppleShit() && _.contains(names, file.name)){
                file.reference = nextRef();

                // Extract file name and extension and add increment
                var sep = file.name.lastIndexOf('.');
                var name = sep ? file.name.substr(0,sep) : file.name;
                var ext = file.name.substr(sep + 1);
                name = name + '.' + file.reference + '.' + ext;

                // file.name is supposed to be read-only ...
                Object.defineProperty(file,"name",{value: name, writable: true});
            }

            // remove duplicate files
            if (_.contains(names, file.name)) return;

            // Set reference to match file id in the response
            if(!file.reference) file.reference = nextRef();

            // Use correct json fields
            file.fileName = file.name;
            file.fileSize = file.size;
            file.fileType = file.type;

            $scope.files.push(file);
        });
    };

    // Kikoo style water drop effect
    $scope.waterDrop = function(event){
        var body = $('body');

        // Create div centered on mouse click event
        var pulse1 = $(document.createElement('div'))
            .css({ left : event.clientX - 50, top : event.clientY - 50 })
            .appendTo(body);
        var pulse2 = $(document.createElement('div'))
            .css({ left : event.clientX - 50, top : event.clientY - 50 })
            .appendTo(body);

        // Add animation class
        pulse1.addClass("pulse1");
        pulse2.addClass("pulse2");

        // Clean after animation
        setTimeout(function(){
            pulse1.remove();
            pulse2.remove();
        },1100);
    };

    // Called when a file is dropped
    $scope.onFileDrop = function(files,event){
        $scope.onFileSelect(files);
        $scope.waterDrop(event);
    };

    // Remove a file from the upload list
    $scope.removeFile = function (file) {
        $scope.files = _.reject($scope.files, function (f) {
            return f.reference == file.reference;
        });
    };

    // Create a new upload
    $scope.newUpload = function () {
        if (!$scope.files.length) return;
        // Get TTL value
        if(!$scope.checkTTL()) return;
        $scope.upload.ttl = $scope.getTTL();
        // HTTP basic auth prompt dialog
        if ($scope.password && !($scope.upload.login && $scope.upload.password)) {
            $scope.getPassword();
            return;
        }
        // Yubikey prompt dialog
        if ($scope.config.yubikeyEnabled && $scope.yubikey && !$scope.upload.yubikey) {
            $scope.getYubikey();
            return;
        }
        // Create file to upload list
        $scope.upload.files = {};
        var ko = _.find($scope.files, function (file) {
            // Check file name length
            if(file.fileName.length > fileNameMaxLength) {
                $dialog.alert({
                    status: 0,
                    message: "File name max length is " + fileNameMaxLength + " characters"
                });
                return true; // break find loop
            }
            // Check invalid characters
            if (!$scope.fileNameValidator(file.fileName)) {
                $dialog.alert({
                    status: 0,
                    message: "Invalid file name " + file.fileName + "\n",
                    value: "Forbidden characters are : " + invalidCharList.join(' ')
                });
                return true; // break find loop
            }
            // Sanitize file object
            $scope.upload.files[file.reference] = {
                fileName : file.fileName,
                fileType : file.fileType,
                fileSize : file.fileSize,
                reference : file.reference
            };
        });
        if(ko) return;
        $api.createUpload($scope.upload)
            .then(function (upload) {
                $scope.upload = upload;
                // Match file metadata using the reference
                _.each($scope.upload.files, function (file) {
                    _.every($scope.files, function (f) {
                        if (f.reference == file.reference) {
                            f.metadata = file;
                            return false;
                        }
                        return true;
                    });
                });
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
            if (!file.metadata.id) return;
            var progress = function (event) {
                // Update progress bar callback
                file.progress = parseInt(100.0 * event.loaded / event.total);
            };
            file.metadata.status = "uploading";
            $api.uploadFile($scope.upload, file, progress, $scope.basicAuth)
                .then(function (metadata) {
                    file.metadata = metadata;
                })
                .then(null, function (error) {
                    $dialog.alert(error);
                });
        });
    };

    // Remove a file from the servers
    $scope.delete = function (file) {
        if (!$scope.upload.uploadToken) return;
        $api.removeFile($scope.upload, file)
            .then(function () {
                $scope.files = _.reject($scope.files, function (f) {
                    return f.metadata.reference == file.metadata.reference;
                });
                // Redirect to main page if no more files
                if (!$scope.files.length) {
                    $location.search('id', null);
                    $route.reload();
                }
            })
            .then(null, function (error) {
                $dialog.alert(error);
            });
    };

    // Compute human readable size
    $scope.humanReadableSize = function (size) {
        if (_.isUndefined(size)) return;
        return filesize(size, {base: 2});
    };

    // Return file download URL
    $scope.getFileUrl = function (file, dl) {
        if (!file || !file.metadata) return;
        var mode = $scope.upload.stream ? "stream" : "file";
        var url = location.origin + '/' + mode + '/' + $scope.upload.id + '/' + file.metadata.id + '/' + file.metadata.fileName;
        if (dl) {
            // Force file download
            url += "?dl=1";
        }

        return url;
    };

    // Return QR Code image url
    $scope.getQrCodeUrl = function (url, size) {
        if (!url) return;
        return location.origin + "/qrcode?url=" + encodeURIComponent(url) + "&size=" + size;
    };

    // Return QR Code image url for current upload
    $scope.getQrCodeUploadUrl = function(size) {
        return $scope.getQrCodeUrl(window.location.href,size);
    };

    // Return QR Code image url for file
    $scope.getQrCodeFileUrl = function(file, size) {
        return $scope.getQrCodeUrl($scope.getFileUrl($scope.qrcode, false),size);
    };

    // Display QR Code dialog for current upload
    $scope.displayQRCodeUpload = function() {
        var url = window.location.href;
        var qrcode = $scope.getQrCodeUrl(url, 400);
        $scope.displayQRCode(url,url,qrcode);
    };

    // Display QR Code dialog for file
    $scope.displayQRCodeFile = function(file) {
        var url = $scope.getFileUrl(file, false);
        var qrcode = $scope.getQrCodeUrl(url, 400);
        $scope.displayQRCode(file.metadata.fileName,url,qrcode);
    };

    // Display QRCode dialog
    $scope.displayQRCode = function(title, url, qrcode) {
        var opts = {
            backdrop: true,
            backdropClick: true,
            templateUrl: 'partials/qrcode.html',
            controller: 'QRCodeController',
            resolve: {
                args: function () {
                    return {
                        title: title,
                        url: url,
                        qrcode: qrcode
                    };
                }
            }
        };
        $dialog.openDialog(opts);
    };

    // Basic auth credentials dialog
    $scope.getPassword = function () {
        var opts = {
            backdrop: true,
            backdropClick: true,
            templateUrl: 'partials/password.html',
            controller: 'PasswordController',
            resolve: {
                args: function () {
                    return {
                        callback: function (login, password) {
                            $scope.upload.login = login;
                            $scope.upload.password = password;
                            $scope.basicAuth = btoa(login + ":" + password);
                            $scope.newUpload();
                        }
                    }
                }
            }
        };
        $dialog.openDialog(opts);
    };

    // Yubikey OTP upload dialog
    $scope.getYubikey = function () {
        var opts = {
            backdrop: true,
            backdropClick: true,
            templateUrl: 'partials/yubikey.html',
            controller: 'YubikeyController',
            resolve: {
                args: function () {
                    return {
                        callback: function (otp) {
                            $scope.upload.yubikey = otp;
                            $scope.newUpload();
                        }
                    }
                }
            }
        };
        $dialog.openDialog(opts);
    };

    // Yubikey OTP download dialog
    $scope.downloadWithYubikey = function (url) {
        var opts = {
            backdrop: true,
            backdropClick: true,
            templateUrl: 'partials/yubikey.html',
            controller: 'YubikeyController',
            resolve: {
                args: function () {
                    return {
                        callback: function (token) {
                            // Redirect to file download URL with yubikey token
                            window.location.replace(url + '/yubikey/' + token);
                        }
                    }
                }
            }
        };
        $dialog.openDialog(opts);
    };

    $scope.ttlUnits = ["days", "hours", "minutes"];
    $scope.ttlUnit = "days";
    $scope.ttlValue = 30;

    // Change ttl unit
    $scope.switchTimeUnit = function () {
        var index = (_.indexOf($scope.ttl.units, $scope.ttl.unit) + 1) % $scope.ttl.units.length;
        $scope.ttl.unit = $scope.ttl.units[index];
    };

    // Return TTL value in seconds
    $scope.getTTL = function () {
        var ttl = $scope.ttlValue;
        if (ttl > 0) {
            if ($scope.ttlUnit == "minutes") {
                ttl = ttl * 60;
            } else if ($scope.ttlUnit == "hours") {
                ttl = ttl * 3600;
            } else if ($scope.ttlUnit == "days") {
                ttl = ttl * 86400;
            }
        } else {
            ttl = -1;
        }
        return ttl;
    };

    // Return TTL unit and value
    $scope.getHumanReadableTTL = function (ttl) {
        var value,unit;
        if (ttl == -1){
            value = -1;
            unit = "never"
        } else if(ttl < 3600){
            value = Math.round(ttl / 60);
            unit = "minutes"
        } else if (ttl < 86400){
            value = Math.round(ttl / 3600);
            unit = "hours"
        } else if (ttl > 86400){
            value = Math.round(ttl / 86400);
            unit = "days"
        } else {
            value = 0;
            unit = "invalid";
        }
        return [value,unit];
    };

    // Check TTL value
    $scope.checkTTL = function() {
        var ok = true;

        // Fix never value
        if ($scope.ttlUnit == 'never') {
            $scope.ttlValue = -1;
        }

        // Get TTL in seconds
        var ttl = $scope.getTTL();

        // Invalid negative value
        if ($scope.ttlUnit != 'never' && ttl < 0) ok = false;
        // Check against server side allowed maximum
        if ($scope.config.maxTTL > 0 && ttl > $scope.config.maxTTL) ok = false;

        if (!ok) {
            var maxTTL = $scope.getHumanReadableTTL($scope.config.maxTTL);
            $dialog.alert({
                status: 0,
                message: "Invalid expiration delay : " + $scope.ttlValue + " " + $scope.ttlUnit,
                value: "Maximum expiration delay is : " + maxTTL[0] + " " + maxTTL[1]
            });
            $scope.setDefaultTTL();
        }

        return ok;
    };

    // Set TTL value to server defaultTTL
    $scope.setDefaultTTL = function(){
        if($scope.config.maxTTL == -1){
            // Never expiring upload is allowed
            $scope.ttlUnits = ["days", "hours", "minutes", "never"];
        }
        var ttl = $scope.getHumanReadableTTL($scope.config.defaultTTL);
        $scope.ttlValue = ttl[0];
        $scope.ttlUnit = ttl[1];
    };

    // Return upload expiration date string
    $scope.getExpirationDate = function () {
        if ($scope.upload.ttl == -1) {
            return "never expire";
        } else {
            var d = new Date(($scope.upload.ttl + $scope.upload.uploadDate) * 1000);
            return "expire the " + d.toLocaleDateString() + " at " + d.toLocaleTimeString();
        }

    };

    // Add upload token in url so one can add/remove files later
    $scope.setAdminUrl = function () {
        $location.search('uploadToken', $scope.upload.uploadToken);
    };

    // Focus the given element by id
    $scope.focus = function(id) {
        angular.element('#'+id)[0].focus();
    };

    $scope.init();
}

// Client download controller
function ClientListCtrl($scope, $api, $dialog) {
    $scope.clients = [];

    $api.getVersion()
        .then(function (buildInfo) {
            $scope.clients = buildInfo.clients;
        })
        .then(null, function (error) {
            $dialog.alert(error);
        });
}

// Alert modal dialog controller
function AlertDialogController($rootScope, $scope, $modalInstance, args) {
    $rootScope.registerDialog($scope);

    $scope.title = 'Success !';
    if (args.data.status != 100) $scope.title = 'Oops !';

    $scope.data = args.data;

    $scope.close = function (result) {
        $rootScope.dismissDialog($scope);
        $modalInstance.close(result);
        if (args.callback) {
            args.callback(result);
        }
    };
}

// HTTP basic auth credentials dialog controller
function PasswordController($rootScope, $scope, $modalInstance, args) {
    $rootScope.registerDialog($scope);

    // Ugly but it works
    setTimeout(function () {
        $("#login").focus();
    }, 100);

    $scope.title = 'Please fill credentials !';
    $scope.login = 'plik';
    $scope.password = '';

    $scope.close = function (login, password) {
        if (!(login.length > 0 && password.length > 0)) {
            return;
        }
        $scope.dismiss();
        if (args.callback) {
            args.callback(login, password);
        }
    };

    $scope.dismiss = function () {
        $rootScope.dismissDialog($scope);
        $modalInstance.close();
    }
}

// Yubikey dialog controller
function YubikeyController($rootScope, $scope, $modalInstance, args) {
    $rootScope.registerDialog($scope);

    // Ugly but it works
    setTimeout(function () {
        $("#yubikey").focus();
    }, 100);

    $scope.title = 'Please fill in a Yubikey OTP !';
    $scope.token = '';

    $scope.check = function (token) {
        if (token.length == 44) {
            $scope.close(token);
        }
    };

    $scope.close = function (result) {
        $scope.dismiss();
        if (args.callback) {
            args.callback(result);
        }
    };

    $scope.dismiss = function () {
        $rootScope.dismissDialog($scope);
        $modalInstance.close();
    }
}

// QRCode dialog controller
function QRCodeController($rootScope, $scope, $modalInstance, args) {
    $rootScope.registerDialog($scope);

    $scope.args = args;

    $scope.close = function () {
        $rootScope.dismissDialog($scope);
        $modalInstance.close();
    };
}