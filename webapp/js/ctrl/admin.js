// Admin controller
plik.controller('AdminCtrl', ['$scope', '$api', '$config', '$dialog', '$location',
    function ($scope, $api, $config, $dialog, $location) {

        $scope.config = {}

        // Get server config
        $config.config
            .then(function (config) {
                // Check if authentication is enabled server side
                if (!config.authentication) {
                    $location.path('/');
                }

                $scope.config = config;

                // Get authenticated user
                $config.getOriginalUser()
                    .then(function (original_user) {
                        $scope.original_user = original_user;

                        // Check if authenticated user is admin
                        if (!original_user.admin) {
                            $location.path('/');
                        }

                        // Get server version
                        $config.getVersion()
                            .then(function (version) {
                                $scope.version = version;
                            })
                            .then(null, function (error) {
                                $dialog.alert(error);
                            });
                    })
                    .then(null, function (error) {
                        $dialog.alert(error);
                    });
            })
            .then(null, function (error) {
                $dialog.alert(error);
            });

        // Display statistics page
        $scope.displayStats = function () {
            $scope.stats = undefined;
            $scope.users = undefined;
            $scope.display = 'stats';

            // Get server statistics
            $config.getServerStats()
                .then(function (stats) {
                    $scope.stats = stats;
                })
                .then(null, function (error) {
                    $dialog.alert(error);
                });
        };

        // Display user management page
        $scope.displayUsers = function (more) {
            if (!more) {
                $scope.stats = undefined;
                $scope.users = [];
                $scope.cursor = undefined;
                $scope.display = 'users';

                // Load possible fake user
                $scope.fake_user = $api.fake_user;
            }

            $scope.limit = 50;

            // Get users
            $api.getUsers($scope.limit, $scope.cursor)
                .then(function (result) {
                    // Success
                    $scope.users = $scope.users.concat(result.results);
                    $scope.cursor = result.after;
                })
                .then(null, function (error) {
                    // Failure
                    $dialog.alert(error);
                });
        };

        // Display create user dialog
        $scope.createUser = function () {
            $dialog.openDialog({
                backdrop: true,
                backdropClick: true,
                templateUrl: 'partials/user.html',
                controller: 'UserController',
                resolve: {
                    args: function () { return {}; }
                }
            }).result.then(
                function (result) {
                    if (result.user) {
                        $api.createUser(result.user)
                            .then(function (user) {
                                $scope.displayUsers();
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

        // Display edit user dialog
        $scope.editUser = function (user) {
            $dialog.openDialog({
                backdrop: true,
                backdropClick: true,
                templateUrl: 'partials/user.html',
                controller: 'UserController',
                resolve: {
                    args: function () { return { user : user }; }
                }
            }).result.then(
                function (result) {
                    if (result.user) {
                        $api.updateUser(result.user)
                            .then(function (user) {
                                $scope.displayUsers();
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

        // Display edit user dialog
        $scope.deleteUser = function (user) {
            $dialog.alert({
                title: "Really ?",
                message: "This will remove " + user.provider + " user " + user.login + " from the server",
                confirm: true
            }).result.then(
                function () {
                    $api.deleteUser(user)
                        .then(function () {
                            $scope.users = _.reject($scope.users, function (u) {
                                return u.id === user.id;
                            });
                        })
                        .then(null, function (error) {
                            $dialog.alert(error);
                        });
                }, function () {
                    // Avoid "Possibly unhandled rejection"
                });
        };

        // This functionality allows an admin to browse another user account
        // In order to delete it or delete some uploads if needed
        $scope.impersonate = function (user) {
            if (!user) {
                // call with no user to cancel the effect
                $scope.setFakeUser(undefined);
                $config.refreshUser();
                return;
            }

            // Don't let users impersonate themselves even if harmless
            if ($scope.original_user.id ===  user.id) return;

            $scope.setFakeUser(user);

            // Dummy try to double-check that we can get the user
            $api.getUser()
                .then(function () {
                    // Success
                    $config.refreshUser();
                })
                .then(null, function (error) {
                    // Failure
                    $dialog.alert(error);
                    $scope.setFakeUser(undefined);
                });
        };

        $scope.setFakeUser = function (user) {
            $api.fake_user = user;

            // We can't call the $api.fake_user from the HTML partial
            $scope.fake_user = user;
        };

        $scope.getUserMaxFileSize = function (user) {
            if (user.maxFileSize > 0) {
                return $scope.humanReadableSize(user.maxFileSize);
            }
            if (user.maxFileSize === 0 && $scope.config.maxFileSize > 0) {
                return "default";
            }
            return "unlimited"
        };

        $scope.getUserMaxUserSize = function (user) {
            if (user.maxUserSize > 0) {
                return $scope.humanReadableSize(user.maxUserSize);
            }
            if (user.maxUserSize === 0 && $scope.config.maxUserSize > 0) {
                return "default";
            }
            return "unlimited"
        };
        
        $scope.getUserMaxTTL = function (user) {
            if (user.maxTTL > 0) {
                return getHumanReadableTTLString(user.maxTTL)
            }
            if (user.maxTTL === 0 && $scope.config.maxTTL > 0) {
                return "default";
            }
            return "unlimited"
        };

        $scope.getHumanReadableTTLString = getHumanReadableTTLString;
        $scope.humanReadableSize = getHumanReadableSize;

        $scope.displayStats();

    }]);