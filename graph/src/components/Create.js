import React from 'react'

import Form from 'react-bootstrap/Form'

import FormControl from 'react-bootstrap/FormControl'

const Create = (props) => (
    <Form inline>
      <FormControl type="text" placeholder="Name" className="mr-sm-2" onChange={e => props.update(e.target.value)} />
    </Form>
)

export default Create