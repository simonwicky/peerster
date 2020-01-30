const express = require('express')
var cors = require('cors')
var bodyParser = require('body-parser')
const app = express()
const { spawn } = require('child_process')
const axios = require('axios')
app.use(cors())
app.use(bodyParser.json())
app.post('/', function (req, res) {
    const children = req.body.cmds.map(cmd => {
        const [prog, ...args] = cmd.split(' ')
        console.info(`spawning ${prog} ${args}`)
        return spawn(prog, args.concat(['-filter', 'rec,init']))
    })
    children.forEach(child => {
        child.stdout.on('data', chunk => console.log(chunk.toString()))
        child.stderr.on('data', line => console.error(line.toString()))
    })
    //children[0].stderr.on('data', line => console.error(line.toString()))
})
app.get('/proxies/:port', async (req, res) => {
    const address = `http://127.0.0.1:${req.params.port}/proxies`
    const proxies = await axios.get(address)
    res.json(proxies.data)

})
app.listen(3000, function () {
  console.log('Example app listening on port 3000!')
})
