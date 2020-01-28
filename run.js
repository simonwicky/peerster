const express = require('express')
var cors = require('cors')
var bodyParser = require('body-parser')
const app = express()
const { spawn } = require('child_process')
app.use(cors())
app.use(bodyParser.json())
app.post('/', function (req, res) {
    
    const children = req.body.cmds.map(cmd => {
        const [prog, ...args] = cmd.split(' ')
        return spawn(prog, args)
    })
    children[0].stdout.on('data', chunk => console.log(chunk.toString()))
})

app.listen(3000, function () {
  console.log('Example app listening on port 3000!')
})
