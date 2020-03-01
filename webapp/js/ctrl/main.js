// Main controller
plik.controller('MainCtrl', ['$scope', '$api', '$config', '$route', '$location', '$dialog',
    function ($scope, $api, $config, $route, $location, $dialog) {

        $scope.sortField = 'metadata.fileName';
        $scope.sortOrder = false;

        $scope.upload = {};
        $scope.files = [];
        $scope.password = false;

        // Get server config
        $config.getConfig()
            .then(function (config) {
                $scope.config = config;
                $scope.setDefaultTTL();
            })
            .then(null, function (error) {
                $dialog.alert(error);
            });

        // File name checks
        var fileNameMaxLength = 1024;
        var invalidCharList = ['/', '#', '?', '%', '"'];
        $scope.fileNameValidator = function (fileName) {
            if (_.isUndefined(fileName)) return false;
            if (fileName.length === 0 || fileName.length > fileNameMaxLength) return false;
            return _.every(invalidCharList, function (char) {
                return fileName.indexOf(char) === -1;
            });
        };

        // Initialize main controller
        $scope.init = function () {
            $scope.mode = 'upload';
            // Display error from redirect if any
            var err = $location.search().err;
            if (!_.isUndefined(err)) {
                var code = $location.search().errcode;
                $dialog.alert({status: code, message: err}).result.then($scope.mainpage);
                return;
            } else {
                // Load current upload id
                $scope.load($location.search().id);
            }
        };

        // Load upload from id
        $scope.load = function (id) {
            if (!id) return;
            $scope.mode = 'download';
            $scope.upload.id = id;
            $scope.upload.uploadToken = $location.search().uploadToken;
            $api.getUpload($scope.upload.id, $scope.upload.uploadToken)
                .then(function (upload) {
                    _.extend($scope.upload, upload);
                    $scope.files = _.map($scope.upload.files, function (file) {
                        return {metadata: file};
                    });

                    // Redirect to home when all stream uploads are downloaded
                    if (!$scope.somethingOk()) {
                        $scope.mainpage();
                    }
                })
                .then(null, function (error) {
                    $dialog.alert(error).result.then($scope.mainpage);
                });
        };

        // Reference is needed to match files ids
        var reference = -1;
        var nextRef = function () {
            reference++;
            return reference.toString();
        };

        // Detect shitty Apple devices
        $scope.isAppleShit = function () {
            return navigator.userAgent.match(/iPhone/i)
                || navigator.userAgent.match(/iPad/i)
                || navigator.userAgent.match(/iPod/i);
        };

        // Add a file to the upload list
        $scope.onFileSelect = function (files) {
            _.each(files, function (file) {
                // Check file size
                if ($scope.config.maxFileSize && file.size > $scope.config.maxFileSize) {
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
                if ($scope.isAppleShit() && _.contains(names, file.name)) {
                    file.reference = nextRef();

                    // Extract file name and extension and add increment
                    var sep = file.name.lastIndexOf('.');
                    var name = sep ? file.name.substr(0, sep) : file.name;
                    var ext = file.name.substr(sep + 1);
                    name = name + '.' + file.reference + '.' + ext;

                    // file.name is supposed to be read-only ...
                    Object.defineProperty(file, "name", {value: name, writable: true});
                }

                // remove duplicate files
                if (_.contains(names, file.name)) return;

                // Set reference to match file id in the response
                if (!file.reference) file.reference = nextRef();

                // Use correct json fields
                file.fileName = file.name;
                file.fileSize = file.size;
                file.fileType = file.type;

                file.metadata = {status: "toUpload"};

                $scope.files.push(file);
            });
        };

        // Kikoo style water drop effect
        $scope.waterDrop = function (event) {
            var body = $('body');

            // Create div centered on mouse click event
            var pulse1 = $(document.createElement('div'))
                .css({left: event.clientX - 50, top: event.clientY - 50})
                .appendTo(body);
            var pulse2 = $(document.createElement('div'))
                .css({left: event.clientX - 50, top: event.clientY - 50})
                .appendTo(body);

            // Add animation class
            pulse1.addClass("pulse1");
            pulse2.addClass("pulse2");

            // Clean after animation
            setTimeout(function () {
                pulse1.remove();
                pulse2.remove();
            }, 1100);
        };

        // Called when a file is dropped
        $scope.onFileDrop = function (files, event) {
            $scope.onFileSelect(files);
            $scope.waterDrop(event);
        };

        // Remove a file from the upload list
        $scope.removeFile = function (file) {
            $scope.files = _.reject($scope.files, function (f) {
                return f.reference === file.reference;
            });
        };

        // Create a new upload
        $scope.newUpload = function (empty) {
            if (!empty && !$scope.files.length) return;
            if ($scope.upload.id) {
                // When adding file to an existing upload
                $scope.uploadFiles();
            } else {
                // Get TTL value
                if (!$scope.checkTTL()) return;
                $scope.upload.ttl = $scope.getTTL();
                // HTTP basic auth prompt dialog
                if ($scope.password && !($scope.upload.login && $scope.upload.password)) {
                    $scope.getPassword();
                    return;
                }

                // Create file to upload list
                $scope.upload.files = [];
                var ko = _.find($scope.files, function (file) {
                    // Check file name length
                    if (file.fileName.length > fileNameMaxLength) {
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
                    $scope.upload.files.push({
                        fileName: file.fileName,
                        fileType: file.fileType,
                        fileSize: file.fileSize,
                        reference: file.reference
                    });
                });
                if (ko) return;

                $api.createUpload($scope.upload)
                    .then(function (upload) {
                        $scope.upload = upload;
                        // Match file metadata using the reference
                        _.each($scope.upload.files, function (file) {
                            _.every($scope.files, function (f) {
                                if (f.reference === file.reference) {
                                    f.metadata = file;
                                    f.metadata.status = "toUpload";
                                    return false;
                                }
                                return true;
                            });
                        });
                        $location.search('id', $scope.upload.id);
                        if (empty) $scope.setAdminUrl();
                        $scope.uploadFiles();
                    })
                    .then(null, function (error) {
                        $dialog.alert(error);
                    });
            }
        };

        // Upload every files
        $scope.uploadFiles = function () {
            if (!$scope.upload.id) return;
            $scope.mode = 'download';
            _.each($scope.files, function (file) {
                if (!(file.metadata && file.metadata.status === "toUpload")) return;
                var progress = function (event) {
                    // Update progress bar callback
                    file.progress = parseInt(100.0 * event.loaded / event.total);
                };
                file.metadata.status = "uploading";
                $api.uploadFile($scope.upload, file, progress, $scope.basicAuth)
                    .then(function (metadata) {
                        file.metadata = metadata;

                        // Redirect to home whe params...n all stream uploads are downloaded
                        if (!$scope.somethingOk()) {
                            $scope.mainpage();
                        }
                    })
                    .then(null, function (error) {
                        file.metadata.status = "toUpload";
                        $dialog.alert(error);
                    });
            });
        };

        // Remove the whole upload
        // Remove a file from the server
        $scope.removeUpload = function () {
            if (!$scope.upload.removable && !$scope.upload.admin) return;

            $dialog.alert({
                title: "Really ?",
                message: "This will remove " + $scope.files.length + " file(s) from the server",
                confirm: true
            }).result.then(
                function () {
                    $api.removeUpload($scope.upload)
                        .then(function () {
                            $scope.mainpage();
                        })
                        .then(null, function (error) {
                            $dialog.alert(error);
                        });
                }, function () {
                    // Avoid "Possibly unhandled rejection"
                });
        };

        // Remove a file from the servers
        $scope.deleteFile = function (file) {
            if (!$scope.upload.removable && !$scope.upload.admin) return;
            $api.removeFile($scope.upload, file)
                .then(function () {
                    $scope.files = _.reject($scope.files, function (f) {
                        return f.metadata.id === file.metadata.id;
                    });
                    // Redirect to main page if no more files
                    if (!$scope.files.length) {
                        $scope.mainpage();
                    }
                })
                .then(null, function (error) {
                    $dialog.alert(error);
                });
        };

        // Check if file is downloadable
        $scope.isDownloadable = function (file) {
            if ($scope.upload.stream) {
                if (file.metadata.status === 'uploading') return true;
            } else {
                if (file.metadata.status === 'uploaded') return true;
            }
            return false;
        };

        // Check if file is in a error status
        $scope.isOk = function (file) {
            if (file.metadata.status === 'toUpload') return true;
            else if (file.metadata.status === 'uploading') return true;
            else if (file.metadata.status === 'uploaded') return true;
            else if ($scope.upload.stream && file.metadata.status === 'missing') return true;
            return false;
        };

        // Is there at least one file ready to be uploaded
        $scope.somethingToUpload = function () {
            return _.find($scope.files, function (file) {
                if (file.metadata.status === "toUpload") return true;
            });
        };

        // Is there at least one file ready to be downloaded
        $scope.somethingToDownload = function () {
            return _.find($scope.files, function (file) {
                return $scope.isDownloadable(file);
            });
        };

        // Is there at least one file not in error
        $scope.somethingOk = function () {
            return _.find($scope.files, function (file) {
                return $scope.isOk(file);
            });
        };

        // Compute human readable size
        $scope.humanReadableSize = function (size) {
            if (_.isUndefined(size)) return;
            return filesize(size, {base: 2});
        };

        $scope.getMode = function () {
            return $scope.upload.stream ? "stream" : "file";
        };

        // Build file download URL
        var getFileUrl = function (mode, uploadID, fileID, fileName, dl) {
            var domain = $scope.config.downloadDomain ? $scope.config.downloadDomain : $api.base;
            var url = domain + '/' + mode + '/' + uploadID;
            if (fileID) {
                url += '/' + fileID;
            }
            if (fileName) {
                url += '/' + fileName;
            }
            if (dl) {
                // Force file download
                url += "?dl=1";
            }

            return encodeURI(url);
        };

        // Return file download URL
        $scope.getFileUrl = function (file, dl) {
            if (!file || !file.metadata) return;
            return getFileUrl($scope.getMode(), $scope.upload.id, file.metadata.id, file.metadata.fileName, dl);
        };

        // Return zip archive download URL
        $scope.getZipArchiveUrl = function (dl) {
            if (!$scope.upload.id) return;
            return getFileUrl("archive", $scope.upload.id, null, "archive.zip", dl);
        };

        // Return QR Code image url
        $scope.getQrCodeUrl = function (url, size) {
            if (!url) return;
            return $api.base + "/qrcode?url=" + encodeURIComponent(url) + "&size=" + size;
        };

        // Return QR Code image url for current upload
        $scope.getQrCodeUploadUrl = function (size) {
            return $scope.getQrCodeUrl(window.location.href, size);
        };

        // Return QR Code image url for file
        $scope.getQrCodeFileUrl = function (file, size) {
            return $scope.getQrCodeUrl($scope.getFileUrl($scope.qrcode, false), size);
        };

        // Display QR Code dialog for current upload
        $scope.displayQRCodeUpload = function () {
            var url = window.location.href;
            var qrcode = $scope.getQrCodeUrl(url, 400);
            $scope.displayQRCode(url, url, qrcode);
        };

        // Display QR Code dialog for file
        $scope.displayQRCodeFile = function (file) {
            var url = $scope.getFileUrl(file, false);
            var qrcode = $scope.getQrCodeUrl(url, 400);
            $scope.displayQRCode(file.metadata.fileName, url, qrcode);
        };

        // Display QRCode dialog
        $scope.displayQRCode = function (title, url, qrcode) {
            $dialog.openDialog({
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
            });
        };

        // Basic auth credentials dialog
        $scope.getPassword = function () {
            $dialog.openDialog({
                backdrop: true,
                backdropClick: true,
                templateUrl: 'partials/password.html',
                controller: 'PasswordController'
            }).result.then(
                function (result) {
                    $scope.upload.login = result.login;
                    $scope.upload.password = result.password;
                    $scope.basicAuth = btoa(result.login + ":" + result.password);
                    $scope.newUpload();
                }, function () {
                    // Avoid "Possibly unhandled rejection"
                });
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
                if ($scope.ttlUnit === "minutes") {
                    ttl = ttl * 60;
                } else if ($scope.ttlUnit === "hours") {
                    ttl = ttl * 3600;
                } else if ($scope.ttlUnit === "days") {
                    ttl = ttl * 86400;
                }
            } else {
                ttl = -1;
            }
            return ttl;
        };

        // Return TTL unit and value
        $scope.getHumanReadableTTL = function (ttl) {
            var value, unit;
            if (ttl === -1) {
                value = -1;
                unit = "never"
            } else if (ttl < 3600) {
                value = Math.round(ttl / 60);
                unit = "minutes"
            } else if (ttl < 86400) {
                value = Math.round(ttl / 3600);
                unit = "hours"
            } else if (ttl > 86400) {
                value = Math.round(ttl / 86400);
                unit = "days"
            } else {
                value = 0;
                unit = "invalid";
            }
            return [value, unit];
        };

        // Check TTL value
        $scope.checkTTL = function () {
            var ok = true;

            // Fix never value
            if ($scope.ttlUnit === 'never') {
                $scope.ttlValue = -1;
            }

            // Get TTL in seconds
            var ttl = $scope.getTTL();

            // Invalid negative value
            if ($scope.ttlUnit !== 'never' && ttl < 0) ok = false;
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
        $scope.setDefaultTTL = function () {
            if ($scope.config.maxTTL === -1) {
                // Never expiring upload is allowed
                $scope.ttlUnits = ["days", "hours", "minutes", "never"];
            }
            var ttl = $scope.getHumanReadableTTL($scope.config.defaultTTL);
            $scope.ttlValue = ttl[0];
            $scope.ttlUnit = ttl[1];
        };

        // Return upload expiration date string
        $scope.getExpirationDate = function () {
            if ($scope.upload.ttl === -1) {
                return "never expire";
            } else {
                var d = new Date($scope.upload.expireAt);
                return "expire on " + d.toLocaleDateString() + " at " + d.toLocaleTimeString();
            }
        };

        // Add upload token in url so one can add/remove files later
        $scope.setAdminUrl = function () {
            $location.search('uploadToken', $scope.upload.uploadToken);
        };

        // Focus the given element by id
        $scope.focus = function (id) {
            angular.element('#' + id)[0].focus();
        };

        // Redirect to main page
        $scope.mainpage = function () {
            $location.search({});
            $location.hash("");
            $route.reload();
        };

        $scope.init();
    }]);