// API Service
angular.module('api', ['ngFileUpload']).factory('$api', function ($http, $q, Upload) {
    var api = {base: window.location.origin + window.location.pathname.replace(/\/$/, '')};

    // Make the actual HTTP call and return a promise
    api.call = function (url, method, params, data, uploadToken) {
        var promise = $q.defer();
        var headers = {};
        if (uploadToken) headers['X-UploadToken'] = uploadToken;
        if (api.fake_user) headers['X-Plik-Impersonate'] = api.fake_user.id;
        $http({
            url: url,
            method: method,
            params: params,
            data: data,
            headers: headers
        })
            .then(function success(resp) {
                promise.resolve(resp.data);
            }, function error(resp) {
                // Format HTTP error return for the dialog service
                var message = resp.data ? resp.data : "Unknown error";
                promise.reject({status: resp.status, message: message});
            });
        return promise.promise;
    };

    // Make the actual HTTP call to upload a file and return a promise
    api.upload = function (url, file, progress_cb, basicAuth, uploadToken) {
        var promise = $q.defer();
        var headers = {};
        if (uploadToken) headers['X-UploadToken'] = uploadToken;
        if (basicAuth) headers['Authorization'] = "Basic " + basicAuth;

        Upload
            .upload({
                url: url,
                method: 'POST',
                file: Upload.rename(file, file.fileName),
                headers: headers
            })
            .then(function success(resp) {
                promise.resolve(resp.data);
            }, function error(resp) {
                // Format HTTP error return for the dialog service
                var message = resp.data ? resp.data : "Unknown error";
                promise.reject({status: resp.status, message: message});
            }, progress_cb);

        return promise.promise;
    };

    // Get upload
    api.getUpload = function (uploadId, uploadToken) {
        var url = api.base + '/upload/' + uploadId;
        return api.call(url, 'GET', {}, {}, uploadToken);
    };

    // Create an upload with current settings
    api.createUpload = function (upload) {
        var url = api.base + '/upload';
        return api.call(url, 'POST', {}, upload);
    };

    // Remove an upload
    api.removeUpload = function (upload) {
        var url = api.base + '/upload/' + upload.id;
        return api.call(url, 'DELETE', {}, {}, upload.uploadToken);
    };

    // Upload a file
    api.uploadFile = function (upload, file, progres_cb, basicAuth) {
        var mode = upload.stream ? "stream" : "file";
        var url;
        if (file.id) {
            url = api.base + '/' + mode + '/' + upload.id + '/' + file.id + '/' + file.fileName;
        } else {
            // When adding file to an existing upload
            url = api.base + '/' + mode + '/' + upload.id;
        }
        return api.upload(url, file, progres_cb, basicAuth, upload.uploadToken);
    };

    // Remove a file
    api.removeFile = function (upload, file) {
        var mode = upload.stream ? "stream" : "file";
        var url = api.base + '/' + mode + '/' + upload.id + '/' + file.id + '/' + file.fileName;
        return api.call(url, 'DELETE', {}, {}, upload.uploadToken);
    };

    // Log in
    api.login = function (provider, login, password) {
        var url = api.base + '/auth/' + provider + '/login';
        if (provider === "local") {
            return api.call(url, 'POST', {}, {login: login, password: password})
        } else {
            return api.call(url, 'GET');
        }
    };

    // Log out
    api.logout = function () {
        var url = api.base + '/auth/logout';
        return api.call(url, 'GET');
    };

    // Get user info
    api.getUser = function () {
        var url = api.base + '/me';
        return api.call(url, 'GET');
    };

    // Create user
    api.createUser = function (user) {
        var url = api.base + '/user';
        return api.call(url, 'POST', {}, user);
    };

    // Update user
    api.updateUser = function (user) {
        var url = api.base + '/user/' + user.id;
        return api.call(url, 'POST', {}, user);
    };

    // Create user
    api.deleteUser = function (user) {
        var url = api.base + '/user/' + user.id;
        return api.call(url, 'DELETE');
    };

    // Get user statistics
    api.getUserTokens = function (limit, cursor) {
        var url = api.base + '/me/token';
        return api.call(url, 'GET', {limit: limit, after: cursor});
    };

    // Get user uploads
    api.getUserUploads = function (token, limit, cursor) {
        var url = api.base + '/me/uploads';
        return api.call(url, 'GET', {token: token, limit: limit, after: cursor});
    };

    // Get user statistics
    api.getUserStats = function () {
        var url = api.base + '/me/stats';
        return api.call(url, 'GET');
    };

    // Remove uploads
    api.deleteUploads = function (token) {
        var url = api.base + '/me/uploads';
        return api.call(url, 'DELETE', {token: token});
    };

    // Delete account
    api.deleteAccount = function () {
        var url = api.base + '/me';
        return api.call(url, 'DELETE');
    };

    // Create a new upload token
    api.createToken = function (comment) {
        var url = api.base + '/me/token';
        return api.call(url, 'POST', {}, {comment: comment});
    };

    // Revoke an upload token
    api.revokeToken = function (token) {
        var url = api.base + '/me/token/' + token;
        return api.call(url, 'DELETE');
    };

    // Get server version
    api.getVersion = function () {
        var url = api.base + '/version';
        return api.call(url, 'GET');
    };

    // Get server config
    api.getConfig = function () {
        var url = api.base + '/config';
        return api.call(url, 'GET');
    };

    // Get server statistics
    api.getServerStats = function () {
        var url = api.base + '/stats';
        return api.call(url, 'GET');
    };

    // Get users
    api.getUsers = function (limit, cursor) {
        var url = api.base + '/users';
        return api.call(url, 'GET', {limit: limit, after: cursor});
    };

    // Get uploads
    api.getUploads = function (limit, cursor, user, token, sort, order) {
        var url = api.base + '/uploads';
        return api.call(url, 'GET', {limit: limit, after: cursor, user: user, token: token, sort: sort, order: order});
    };

    return api;
});