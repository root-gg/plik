module.exports = function(grunt) {
    require('load-grunt-tasks')(grunt);

    grunt.initConfig({
        clean: ['public'],
        concat: {
            js_vendors: {
                src: [
                    "bower_components/jquery/dist/jquery.js",
                    "bower_components/bootstrap/dist/js/bootstrap.js",
                    "bower_components/danialfarid-angular-file-upload/dist/angular-file-upload-html5-shim.js",
                    "bower_components/angular/angular.js",
                    "bower_components/angular-sanitize/angular-sanitize.min.js",
                    "bower_components/danialfarid-angular-file-upload/dist/angular-file-upload.js",
                    "bower_components/angular-route/angular-route.js",
                    "bower_components/angular-bootstrap/ui-bootstrap-tpls.js",
                    "bower_components/ng-clip/dest/ng-clip.min.js",
                    "bower_components/angular-markdown-directive/markdown.js",
                    "bower_components/underscore/underscore.js",
                    "bower_components/filesize/lib/filesize.js",
                    "bower_components/zeroclipboard/dist/ZeroClipboard.js",
                    "bower_components/angular-contenteditable/angular-contenteditable.js",
                    "bower_components/showdown/src/showdown.js",
                ],
                dest: 'public/js/vendor.js',

            },
            css_vendors: {
                src: [
                    "bower_components/bootstrap/dist/css/bootstrap.css",
                    "bower_components/fontawesome/css/font-awesome.css",
                    "bower_components/bootstrap-switch/dist/css/bootstrap3/bootstrap-switch.css",
                ],
                dest: 'public/css/vendor.css'
            }
        },
        copy: {
            stylesheets: {
                files: [{
                        expand: true,
                        src: [
                            'bower_components/bootstrap/fonts/*',
                            'bower_components/fontawesome/fonts/fontawesome-webfont.woff',
                            'bower_components/fontawesome/fonts/fontawesome-webfont.tff',
                        ],
                        dest: 'public/fonts/',
                        flatten: true
                }]
            }
        },
        ngAnnotate: {
            options: {
                singleQuotes: true
            },
            all: {
                files: {
                    'public/js/vendor.js': ['public/js/vendor.js']
                }
            }
        },
        uglify: {
            options: {
                mangle: true,
                compress: true,
                report: true,
                sourceMap: true
            },
            javascript: {
                files: {
                    'public/js/vendor.js': ['public/js/vendor.js'],
                }
            }

        },
        cssmin: {
            options: {
                keepSpecialComments : 0
            },
            combine: {
                files: {
                    'public/css/vendor.css': ['public/css/vendor.css']
                }
            }
        }
    });

    grunt.registerTask('default', ['clean', 'concat','copy', 'ngAnnotate', 'uglify', 'cssmin']);
};