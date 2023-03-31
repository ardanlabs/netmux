import {w2grid, w2layout, w2toolbar} from './lib/w2ui/w2ui.js'
import * as loginPopup from "./login_popup.js"
// widget configuration


let config = {
    toolbar: {
        name: 'toolbar',
        items: [
            {type: 'html', id: 'logo', html: `<div><img src="duck-b.png" style="height: 30px; width: auto"></div>`},
            {type: 'html', id: 'title', html: `<div class="w2ui-tb-text"><b>Netmux</b></div>`},
            {type: 'button', id: 'login', text: 'Login Context'},
            {type: 'button', id: 'logout', text: 'Logout Context'},
            {type: 'break'},
            {type: 'button', id: 'start', text: 'Start Context'},
            {type: 'button', id: 'stop', text: 'Stop Context'},
            {type: 'break'},
            {type: 'button', id: 'start_svc', text: 'Start Service'},
            {type: 'button', id: 'stop_svc', text: 'Stop Service'},
            {type: 'break'},
            {type: 'button', id: 'refresh', text: 'Refresh'},
            {type: 'break'},
            {type: 'button', id: 'restart', text: 'Restart'},
        ],
        onClick(event) {
            console.log('Target: ' + event.target, event)
            toolbarHandler[event.target]()

        }
    },
    layout: {
        name: 'layout',
        padding: 0,
        panels: [
            {type: 'top', size: 40, resizable: false, minSize: 40},
            {type: 'main', minSize: 550, overflow: 'hidden'}
        ]
    },
    grid1: {
        name: 'grid1',
        columns: [
            {field: 'ctx', text: 'Context', size: '180px'},
            {field: 'name', text: 'Name', size: '180px'},
            {field: 'status', text: 'Status', size: '80px'},
            {field: 'localaddr', text: 'L. Addr', size: '250px'},
            {field: 'localport', text: 'L. Port', size: '80px'},
            {field: 'remoteaddr', text: 'R. Addr', size: '250px'},
            {field: 'remoteport', text: 'R. Port', size: '80px'},
            {field: 'proto', text: 'Proto', size: '80px'},
            {field: 'nconns', text: 'Conns', size: '80px'},
            {field: 'sent', text: 'Sent', size: '80px'},
            {field: 'recv', text: 'Recv', size: '80px'},

        ],
        records: [],
        async onSelect(event) {
            await event.complete
            selected = event.detail.recid
            //console.log('select', event.detail, this.getSelection());
        }
    },
}

setInterval(refresh, 1000)

/**
 * @type string
 */
let selected = ""

async function refresh() {
    let st = await nx_status()
    if (st.err) {
        console.log(st.err)
        alert(st.err)
        return
    }
    let rows = []
    for (let ctx of st.data.contexts) {
        rows.push({recid: ctx.name, ctx: ctx.name, name: ctx.name})
        if (ctx.services) {
            for (let svc of ctx.services) {
                rows.push({
                    recid: ctx.name + "." + svc.name,
                    ctx: ctx.name,
                    name: svc.name,
                    localaddr: svc.localaddr,
                    localport: svc.localport,
                    remoteaddr: svc.remoteaddr,
                    remoteport: svc.remoteport,
                    proto: svc.proto,
                    status: svc.status,
                    nconns: svc.nconns,
                    sent: svc.sent,
                    recv: svc.recv
                })
            }
        }
    }
    selected = grid1.getSelection()
    grid1.clear()
    grid1.add(rows)
    grid1.select(selected)

}

async function start() {


}

async function stop() {

}


let toolbarHandler = {
    "refresh": refresh,
    "start": async () => {
        let parts = selected.split(".")
        nx_start(parts[0])
    },
    "stop": async () => {
        let parts = selected.split(".")
        nx_stop(parts[0])
    },
    "start_svc": async () => {
        let parts = selected.split(".")
        nx_svc_start(parts[0], parts[1])
    },
    "stop_svc": async () => {
        let parts = selected.split(".")
        nx_svc_stop(parts[0], parts[1])
    },
    "login": () => {
        loginPopup.Show(selected)
    },
    "logout": async () => {
        let parts = selected.split(".")
        let ret = await nx_logout(parts[0])
        if (ret.err) {
            alert(ret.err)
        }
    },
    "restart": async () => {
        let ret = await nx_exit()
        if (ret.err) {
            alert(ret.err)
        }

    }
}

window.addEventListener('resize', function (event) {
    grid1.resize()
}, true);

let layout = new w2layout(config.layout)
let grid1 = new w2grid(config.grid1)
let toolbar = new w2toolbar(config.toolbar)
// initialization
layout.render('#main')
layout.html('top', toolbar)
layout.html('main', grid1)

async function main() {
}

main()
