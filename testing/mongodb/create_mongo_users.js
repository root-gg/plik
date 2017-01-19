use admin;
db.createUser({user:"admin", pwd:"secret", roles:["root"]});
db.auth("admin","secret");
use plik;
db.createUser({user:"plik", pwd:"password", roles:["readWrite"]});
exit;
