// Home controller
plik.controller('HomeCtrl', ['$scope', '$api', '$config', '$dialog', '$location',
    function ($scope, $api, $config, $dialog, $location) {

        $scope.display = 'uploads';
        $scope.displayUploads = function (token) {
            $scope.uploads = [];
            $scope.token = token;
            $scope.display = 'uploads';
            $scope.refreshUser();
        };

        $scope.displayTokens = function () {
            $scope.display = 'tokens';
            $scope.refreshUser();
        };

        // Get server config
        $config.config
            .then(function (config) {
                // Check if authentication is enabled server side
                if (!config.authentication) {
                    $location.path('/');
                }
            })
            .then(null, function (error) {
                $dialog.alert(error);
            });

        // Handle user promise
        var loadUser = function (promise) {
            promise.then(function (user) {
                $scope.user = user;
                $scope.getUploads();
                $scope.getTokens();
                $scope.getUserStats();
                $scope.fake_user = $api.fake_user;
            })
                .then(null, function (error) {
                    if (error.status === 401 || error.status === 403) {
                        $location.path('/login');
                    } else {
                        $dialog.alert(error);
                    }
                });
        };

        // Refresh user
        $scope.refreshUser = function () {
            loadUser($config.refreshUser());
        };

        // page size
        $scope.limit = 50;

        // Get user upload list
        $scope.getUploads = function (more) {
            if (!more) {
                $scope.uploads = [];
                $scope.upload_cursor = undefined;
            }

            // Get user uploads
            $api.getUserUploads($scope.token, $scope.limit, $scope.upload_cursor)
                .then(function (result) {
                    $scope.uploads = $scope.uploads.concat(result.results);
                    $scope.upload_cursor = result.after;
                })
                .then(null, function (error) {
                    $dialog.alert(error);
                });
        };

        // Get user upload list
        $scope.getTokens = function (more) {
            if (!more) {
                $scope.tokens = [];
                $scope.tokens_cursor = undefined;
            }

            // Get user uploads
            $api.getUserTokens($scope.limit, $scope.tokens_cursor)
                .then(function (result) {
                    $scope.tokens = $scope.tokens.concat(result.results);
                    $scope.tokens_cursor = result.after;
                })
                .then(null, function (error) {
                    $dialog.alert(error);
                });
        };

        // Get user statistics
        $scope.getUserStats = function () {
            $api.getUserStats()
                .then(function (stats) {
                    $scope.user.stats = stats;
                })
                .then(null, function (error) {
                    $dialog.alert(error);
                });
        };

        // Remove an upload
        $scope.deleteUpload = function (upload) {
            $dialog.alert({
                title: "Really ?",
                message: "This will remove " + upload.files.length + " file(s) from the server",
                confirm: true
            }).result.then(
                function () {
                    $api.removeUpload(upload)
                        .then(function () {
                            $scope.uploads = _.reject($scope.uploads, function (u) {
                                return u.id === upload.id;
                            });
                        })
                        .then(null, function (error) {
                            $dialog.alert(error);
                        });
                }, function () {
                    // Avoid "Possibly unhandled rejection"
                });
        };

        // Delete all user uploads
        $scope.deleteUploads = function () {
            $dialog.alert({
                title: "Really ?",
                message: "This will remove all uploads from the server",
                confirm: true
            }).result.then(
                function () {
                    $api.deleteUploads($scope.token)
                        .then(function (result) {
                            $scope.uploads = [];
                            $scope.getUploads();
                            $dialog.alert(result);
                        })
                        .then(null, function (error) {
                            $dialog.alert(error);
                        });
                }, function () {
                    // Avoid "Possibly unhandled rejection"
                });
        };

        // Generate a new token
        $scope.createToken = function (comment) {
            $api.createToken(comment)
                .then(function () {
                    $scope.refreshUser();
                })
                .then(null, function (error) {
                    $dialog.alert(error);
                });
        };

        // Revoke a token
        $scope.revokeToken = function (token) {
            $dialog.alert({
                title: "Really ?",
                message: "Revoking a token will not delete associated uploads.",
                confirm: true
            }).result.then(
                function () {
                    $api.revokeToken(token.token)
                        .then(function () {
                            $scope.refreshUser();
                        })
                        .then(null, function (error) {
                            $dialog.alert(error);
                        });
                }, function () {
                    // Avoid "Possibly unhandled rejection"
                });
        };

        // Log out
        $scope.logout = function () {
            $api.logout()
                .then(function () {
                    $config.refreshUser();
                    $location.path('/');
                })
                .then(null, function (error) {
                    $dialog.alert(error);
                });
        };

        // Sign out
        $scope.deleteAccount = function () {
            $dialog.alert({
                title: "Really ?",
                message: "Deleting your account will not delete your uploads.",
                confirm: true
            }).result.then(
                function () {
                    $api.deleteAccount()
                        .then(function () {
                            $config.refreshUser();
                            $location.path('/');
                        })
                        .then(null, function (error) {
                            $dialog.alert(error);
                        });
                }, function () {
                    // Avoid "Possibly unhandled rejection"
                }
            );
        };

        // Edit user
        $scope.editAccount = function () {
            $dialog.openDialog({
                backdrop: true,
                backdropClick: true,
                templateUrl: 'partials/user.html',
                controller: 'UserController',
                resolve: {
                    args: function () { return { user : $scope.user }; }
                }
            }).result.then(
                function (result) {
                    if (result.user) {
                        $api.updateUser(result.user)
                            .then(function (user) {
                            })
                            .then(null, function (error) {
                                $dialog.alert(error);
                            });
                    } else if (result.error) {
                        $dialog.alert(result.error);
                    }
                }, function () {
                    // Avoid "Possibly unhandled rejection"
                });
        };

        // Get upload url
        $scope.getUploadUrl = function (upload) {
            return $api.base + '/#/?id=' + upload.id;
        };

        // Get file url
        $scope.getFileUrl = function (upload, file) {
            return $api.base + '/file/' + upload.id + '/' + file.id + '/' + file.fileName;
        };

        // Compute human readable size
        $scope.humanReadableSize = getHumanReadableSize;

        // Redirect to main page
        $scope.mainpage = function () {
            $location.search({});
            $location.hash("");
            $location.path('/');
        };

        loadUser($config.getUser());
    }]);