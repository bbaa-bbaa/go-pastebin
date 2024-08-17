module.exports = {
	globDirectory: '.',
	globPatterns: [
		'**/*.{js,css,ico}',
    'manifest.json',
    'static/mdui/fonts/roboto/{Roboto-Regular.woff2,Roboto-Medium.woff2,Roboto-Bold.woff2}',
    'static/mdui/icons/material-icons/MaterialIcons-Regular.woff2',
    'static/font/Bender/Bender-Bold.woff2',
    'static/font/Hack/hack-regular.woff2',
	],
  /*
  globIgnores:[
    'index.html',
    'admin.html'
  ],
  */
  skipWaiting: true,
  clientsClaim: true,
	swDest: 'sw.js',
	ignoreURLParametersMatching: [
		/^utm_/,
		/^fbclid$/,
	],
  mode: "production",
  runtimeCaching: [
    {
      urlPattern: function(options){
        const url = options.url;
        return url.pathname.startsWith('/static/img/') || url.pathname.startsWith('/static/') && (url.pathname.endsWith('.js') || url.pathname.endsWith('.css') || url.pathname.endsWith('.woff') || url.pathname.endsWith('.woff2'));
      },
      handler: 'StaleWhileRevalidate'
    },
    {
      urlPattern: function(options){
        const url = options.url;
        return url.pathname === '/' || url.pathname === '/api/user' || url.pathname === '/admin/';
      },
      handler: 'NetworkFirst',
    }
  ],
};
