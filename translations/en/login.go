package english

import (
	"github.com/rmanna/wash/helpers/trans"
)

func login(en ut.Translator) {

	trans.Add(en, "login-login", "Login", false)
	trans.Add(en, "login-welcome", "Welcome {0}", false)
}
