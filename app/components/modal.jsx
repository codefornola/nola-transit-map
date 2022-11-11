import React, { useState } from 'react';
import Button from 'react-bootstrap/Button';


function CustomModal(props) {
    const [show, setShow] = useState(false);
  
    const handleClose = () => setShow(false);
    const handleShow = () => setShow(true);

    return(
        <div>
            <button onClick={handleShow} className="about-button">{props.buttonText}</button>
            {show ?(
            <div class="Modal">
                <aside class="Modal__content">
                    <div class="Modal__content--header">
                        <h2>{props.title}</h2>
                        <p>{props.subtitle}</p>
                    </div>
                        <p>
                            {props.content}
                        </p>
                    <div class="Modal__content--footer">
                        <Button onClick={handleClose}>
                        Close
                        </Button>
                    </div>
                </aside>
            </div>
            ) : (
                null
            )}
        </div>  
    )
}

export default CustomModal;