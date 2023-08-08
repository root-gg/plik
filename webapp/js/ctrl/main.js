// Main controller
plik.controller('MainCtrl', ['$scope', '$api', '$config', '$route', '$location', '$dialog', '$timeout', '$paste', '$q',
    function ($scope, $api, $config, $route, $location, $dialog, $timeout, $paste, $q) {
        var discard = function (e) {
            // Avoid "Possibly unhandled rejection"
        };

        $scope.mode = 'upload';
        $scope.sortField = 'fileName';
        $scope.sortOrder = false;

        $scope.user = null;
        $scope.config = null;

        $scope.upload = {};
        $scope.files = [];
        $scope.password = false;
        $scope.enableComments = false;

        // File name checks
        var fileNameMaxLength = 1024;
        var invalidCharList = ['/', '#', '?', '%', '"'];

        $scope.isFeatureEnabled = function (feature_name) {
            if (!$scope.config) return false;
            var value = $scope.config["feature_" + feature_name];
            return value && value !== "disabled";
        }

        $scope.isFeatureDefault = function (feature_name) {
            if (!$scope.config) return false;
            var value = $scope.config["feature_" + feature_name];
            return value === "default" || value === "forced";
        }

        $scope.isFeatureForced = function (feature_name) {
            if (!$scope.config) return false;
            var value = $scope.config["feature_" + feature_name];
            return value === "forced";
        }

        // Get server config
        $scope.configReady = $q.defer();
        $config.getConfig()
            .then(function (config) {
                $scope.config = config;

                $scope.upload.oneShot = $scope.isFeatureDefault("one_shot")
                $scope.upload.removable = $scope.isFeatureDefault("removable")
                $scope.upload.stream = $scope.isFeatureDefault("stream")
                $scope.password = $scope.isFeatureDefault("password")
                $scope.enableComments = $scope.isFeatureDefault("comments")

                $scope.configReady.resolve(true);
            })
            .then(null, function (error) {
                $dialog.alert(error).result.then($scope.mainpage);
            });

        // Get user
        $scope.userReady = $q.defer();
        $config.getUser()
            .then(function (user) {
                $scope.user = user;
            })
            .then(null, function (error) {
                if (error.status !== 401 && error.status !== 403) {
                    $dialog.alert(error).result.then($scope.mainpage);
                }
            })
            .finally(function () {
                $scope.userReady.resolve(true);
            });

        // Initialize main controller
        $scope.loaded = $q.defer();
        $scope.load = function () {
            // Display error from redirect if any
            var err = $location.search().err;
            if (!_.isUndefined(err)) {
                var code = $location.search().errcode;
                $dialog.alert({status: code, message: err}).result.then($scope.mainpage);
                return;
            }

            // Load current upload id
            var id = $location.search().id
            if (!id) {
                $scope.loaded.resolve(true);
                return
            }

            $scope.mode = 'download';
            $scope.upload.id = id;
            $scope.upload.uploadToken = $location.search().uploadToken;
            $api.getUpload($scope.upload.id, $scope.upload.uploadToken)
                .then(function (upload) {
                    _.extend($scope.upload, upload);
                    $scope.files = $scope.upload.files;

                    // Redirect to home when all stream uploads are downloaded
                    //if (!$scope.somethingOk()) {
                    //    $scope.mainpage();
                    //}

                    $scope.loaded.resolve(true);
                })
                .then(null, function (error) {
                    $dialog.alert(error).result.then($scope.mainpage).then($scope.mainpage);
                });
        };
        $scope.load();

        // whenReady ensure that the scope has been initialized especially :
        // $scope.config, $scope.user, $scope.mode, $scope.upload, $scope.files, ...
        $scope.ready = $q.all([$scope.configReady.promise, $scope.userReady.promise, $scope.loaded.promise]);

        function whenReady(f) {
            $scope.ready.then(f, discard);
        }

        // Redirect to login page if user is not authenticated
        whenReady(function () {
            if ($scope.isFeatureForced("authentication") && $scope.mode === 'upload' && !($scope.user || $scope.uploadToken)) {
                $location.path('/login');
            }
            $scope.setDefaultTTL();
        });

        // Validate that file name is valid
        $scope.fileNameValidator = function (fileName) {
            if (_.isUndefined(fileName)) return false;
            if (fileName.length === 0 || fileName.length > fileNameMaxLength) return false;
            return _.every(invalidCharList, function (char) {
                return fileName.indexOf(char) === -1;
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

        $scope.checkMaxFileSize = function (size) {
            var maxFileSize = $scope.config.maxFileSize;
            if ($scope.user && $scope.user.maxFileSize > 0) {
                maxFileSize = $scope.user.maxFileSize;
            }
            if (maxFileSize && size > maxFileSize) {
                $dialog.alert({
                    status: 0,
                    message: "File is too big : " + getHumanReadableSize(size),
                    value: "Maximum allowed size is : " + getHumanReadableSize($scope.config.maxFileSize)
                });
                return false;
            }
            return true;
        }

        // Add a file to the upload list
        $scope.onFileSelect = function (files) {
            _.each(files, function (file) {
                // Check file size
                if (!$scope.checkMaxFileSize(file.size)) return;

                // Already added file names
                var names = _.pluck($scope.files, 'fileName');

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
                file.status = "toUpload";

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
                $scope.upload.ttl = getTTL($scope.ttlValue, $scope.ttlUnit);
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
                        // Match file using the reference
                        _.each($scope.upload.files, function (file) {
                            _.every($scope.files, function (f) {
                                if (f.reference === file.reference) {
                                    _.extend(f, file);
                                    f.status = "toUpload";
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
                if (file.status !== 'toUpload') return;
                var progress = function (event) {
                    // Update progress bar callback
                    file.progress = parseInt(100.0 * event.loaded / event.total);
                };
                file.status = "uploading";
                $api.uploadFile($scope.upload, file, progress, $scope.basicAuth)
                    .then(function (result) {
                        _.extend(file, result);

                        // Redirect to home whe params...n all stream uploads are downloaded
                        if (!$scope.somethingOk()) {
                            $scope.mainpage();
                        }
                    })
                    .then(null, function (error) {
                        file.status = "toUpload";
                        $dialog.alert(error);
                    });
            });
        };

        // Remove the whole upload
        // Remove a file from the servers
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
                }, discard);
        };

        // Remove a file from the servers
        $scope.deleteFile = function (file) {
            if (!$scope.upload.removable && !$scope.upload.admin) return;

            $dialog.alert({
                title: "Really ?",
                message: "This will remove 1 file from the server",
                confirm: true
            }).result.then(
                function () {
                    $api.removeFile($scope.upload, file)
                        .then(function () {
                            $scope.files = _.reject($scope.files, function (f) {
                                return f.id === file.id;
                            });
                            // Redirect to main page if no more files
                            if (!$scope.files.length) {
                                $scope.mainpage();
                            }
                        })
                        .then(null, function (error) {
                            $dialog.alert(error);
                        })
                }, discard);
        };

        // Check if file is downloadable
        $scope.isDownloadable = function (file) {
            if ($scope.upload.stream) {
                if (file.status === 'uploading') return true;
            } else {
                if (file.status === 'uploaded') return true;
            }
            return false;
        };

        // Check if file is in a error status
        $scope.isOk = function (file) {
            if (file.status === 'toUpload') return true;
            else if (file.status === 'uploading') return true;
            else if (file.status === 'uploaded') return true;
            else if ($scope.upload.stream && file.status === 'missing') return true;
            return false;
        };

        // Is there at least one file ready to be uploaded
        $scope.somethingToUpload = function () {
            return _.find($scope.files, function (file) {
                if (file.status === "toUpload") return true;
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

        // Is it possible to add files to the upload
        $scope.okToAddFiles = function () {
            if ($scope.mode === "upload") return true;
            if ($scope.upload.stream) return false;
            return $scope.upload.admin;
        }


        // Compute human readable size
        $scope.humanReadableSize = getHumanReadableSize;

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
            if (!file || !file.id || !file.fileName) return;
            return getFileUrl($scope.getMode(), $scope.upload.id, file.id, file.fileName, dl);
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
            $scope.displayQRCode(file.fileName, url, qrcode);
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
                }, discard);
        };

        $scope.ttlUnits = ["days", "hours", "minutes"];
        $scope.ttlUnit = "days";
        $scope.ttlValue = 30;

        // Check TTL value
        $scope.checkTTL = function () {
            var ok = true;

            // Fix unlimited value
            if ($scope.ttlUnit === 'unlimited') {
                $scope.ttlValue = -1;
            }

            // Get TTL in seconds
            var ttl = getTTL($scope.ttlValue, $scope.ttlUnit);

            // Invalid negative value
            if ($scope.ttlUnit !== 'unlimited' && ttl < 0) ok = false;

            // Check against server side allowed maximum
            maxTTL = $scope.config.maxTTL;
            if ($scope.user && $scope.user.maxTTL !== 0) {
                maxTTL = $scope.user.maxTTL;
            }

            if (maxTTL > 0 && ttl > maxTTL) ok = false;

            if (!ok) {
                var maxTTL = getHumanReadableTTL(maxTTL);
                $dialog.alert({
                    status: 0,
                    message: "Invalid expiration delay : " + $scope.ttlValue + " " + $scope.ttlUnit + ". " +
                        "Maximum is : " + maxTTL[0] + " " + maxTTL[1],
                });
                $scope.setDefaultTTL();
            }

            return ok;
        };

        // Set TTL value to server defaultTTL
        $scope.setDefaultTTL = function () {
            maxTTL = $scope.config.maxTTL;
            if ($scope.user && $scope.user.maxTTL !== 0) {
                maxTTL = $scope.user.maxTTL;
            }
            if (maxTTL < 0) {
                // Never expiring upload is allowed
                $scope.ttlUnits = ["days", "hours", "minutes", "unlimited"];
            }
            if ($scope.user && $scope.user.maxTTL > 0 && $scope.config.defaultTTL > $scope.user.maxTTL) {
                // If user maxTTL is less than defaultTTL then set to user maxTTL to avoid error on upload
                $scope.config.defaultTTL = $scope.user.maxTTL;
            }
            var ttl = getHumanReadableTTL($scope.config.defaultTTL);
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

        // Display the admin URL link only if the upload token is available and already present in the url
        $scope.displayAdminUrlLink = function () {
            return $scope.upload.uploadToken && !$location.search().uploadToken;
        }

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

        // Called on paste event
        $scope.pasteCallback = function (clipboard) {
            // Dismiss paste event if we can't add files to the upload
            if (!$scope.okToAddFiles()) return;

            // Dismiss paste event if a modal/dialog is already open
            if (document.getElementsByClassName('modal-open').length > 0) return;

            // If clipboard contains files
            if (!$scope.isFeatureForced('text')) {
                var files = _.clone(clipboard.files);
                if (files.length) {
                    // Add the files
                    $timeout(function () {
                        $scope.onFileSelect(files);
                    })
                    return
                }
            }

            // If clipboard contains text
            if ($scope.isFeatureEnabled('text')) {
                var text = clipboard.getData('text');
                if (text) {
                    $scope.openTextDialog(text);
                }
            }
        };

        $scope.openTextDialog = function (text) {
            // Open a dialog to enter text
            $dialog.openDialog({
                backdrop: true,
                backdropClick: true,
                templateUrl: 'partials/paste.html',
                controller: 'PasteController',
                size: 'lg', // large size
                resolve: {
                    args: function () {
                        return {
                            text: text,
                        };
                    }
                }
            }).result.then(
                function (result) {
                    if (result.text) {
                        var filename = 'paste.txt'

                        // Increment filename if already present
                        var names = _.pluck($scope.files, 'fileName');
                        var i = 1;
                        while (_.contains(names, filename)) {
                            filename = 'paste.' + i + '.txt';
                            i++;
                        }

                        // Create a file from the pasted text
                        var blob = new Blob([result.text], {type: "text/plain;charset=utf-8"});
                        var file = new File([blob], filename, {type: "text/plain;charset=utf-8"});

                        // Add the file
                        $timeout(function () {
                            $scope.onFileSelect([file]);
                        })
                    }

                }, discard);
        }

        // Register paste handler
        $scope.registerPasteHandler = function () {
            // Register paste callback to the paste service
            $paste.register($scope.pasteCallback);

            // Unregister paste callback when route changes
            $scope.$on('$routeChangeStart', $paste.unregister);
        };

        whenReady(function () {
            $scope.registerPasteHandler();
            if ($scope.isFeatureDefault("text")) {
                $scope.openTextDialog();
            }
        })
    }
]);