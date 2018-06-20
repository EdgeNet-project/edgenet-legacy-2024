yourFolder = '/Users/rick/Projects/GoogleAppEngine/google-cloud-sdk/platform'
sys.path.insert(1, yourFolder +'/google_appengine')
sys.path.insert(1, yourFolder + '/google_appengine/lib/yaml/lib')
if 'google' in sys.modules:           
    del sys.modules['google']
