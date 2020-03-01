// Admin controller
plik.controller('AdminCtrl', ['$scope', '$api', '$config', '$dialog', '$location',
    function ($scope, $api, $config, $dialog, $location) {

        // Get server config
        $config.config
            .then(function (config) {
                // Check if authentication is enabled server side
                if (!config.authentication) {
                    $location.path('/');
                }

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

        // This functionality allows an admin to browse another user account
        // In order to delete it or delete some uploads if needed
        $scope.impersonate = function (user) {
            if (!user) {
                // call with no user to cancel the effect
                $scope.setFakeUser(undefined);
                $config.refreshUser();
                return;
            }

            $scope.setFakeUser(user);

            // Dummy try to double check that we can get the user
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

        // Compute human readable size
        // TODO This should be global as we also use it in other controllers
        $scope.humanReadableSize = function (size) {
            if (_.isUndefined(size)) return;
            return filesize(size, {base: 2});
        };

        $scope.displayStats();

    }]);