// Client download controller
plik.controller('ClientListCtrl', ['$scope', '$api', '$dialog',
    function ($scope, $api, $dialog) {

        $scope.clients = [];

        $api.getVersion()
            .then(function (buildInfo) {
                $scope.clients = buildInfo.clients;
            })
            .then(null, function (error) {
                $dialog.alert(error);
            });

        $scope.getClientPath = function (client) {
            return $api.base + client.path;
        }
    }]);
