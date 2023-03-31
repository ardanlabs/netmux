import {w2alert, w2form, w2popup, w2ui} from "./lib/w2ui/w2ui.js";

let popup = null

export function Show(selected) {

    let parts = selected.split(".")

    if (!w2ui.foo) {
        new w2form({
            name: 'foo',
            style: 'border: 0px; background-color: transparent;',
            fields: [
                {field: 'context', type: 'text', required: true, html: {label: 'Context'}},
                {field: 'username', type: 'text', required: true, html: {label: 'User Name'}},
                {field: 'password', type: 'password', required: true, html: {label: 'Password'}},

            ],
            record: {
                context: selected,
                username: 'nx',
                password: 'nx'
            },

            actions: {

                async Login() {
                    this.validate()

                    let ret = await nx_login(
                        w2ui.foo.getValue("context"),
                        w2ui.foo.getValue("username"),
                        w2ui.foo.getValue("password"),
                    )


                    setTimeout(async () => {
                        if (popup) {
                            popup.close()
                            popup = null
                        }
                        if (ret.err != null) {
                            w2alert(ret.err)
                        } else {
                            w2alert(`Logged in w success`)
                        }
                    }, 100)


                },
            }
        });
    }
    if (popup == null) {
        popup = w2popup.open({
            name: "loginPopup",
            title: 'Login Credentials',
            body: '<div id="form" style="width: 100%; height: 100%;"></div>',
            style: 'padding: 15px 0px 0px 0px',
            width: 500,
            height: 280,
            showMax: true,
            onClose: function () {
                popup = null
            },
            async onToggle(event) {
                if (e && e.complete) {
                    await event.complete
                }
                w2ui.foo.resize();
            }
        })
            .then((event) => {
                w2ui.foo.render('#form')
            });
    }
}
