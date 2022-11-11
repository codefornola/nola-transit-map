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
            <div class="Modal" onClick={handleClose}>
                <aside class="Modal__content">
                    <div class="Modal__content--header">
                        <h2>{props.title}</h2>
                        <p>{props.subtitle}</p>
                    </div>
                        <p>
                            When the RTA switched to the new LePass app, all of the realtime data
                            stopped working. Relying on public transportation in New Orleans without this data is extremely challenging.
                            We made this map as a stop gap until realtime starts working again.

                            If you find an problem, or have a feature request, consider <a href="https://github.com/codefornola/nola-transit-map/issues">filing an issue here</a>.
                            You can also join us on slack in the #civic-hacking channel of the <a href="https://join.slack.com/t/nola/shared_invite/zt-4882ja82-iGm2yO6KCxsi2aGJ9vnsUQ">Nola Devs slack</a>.

                            Take a look at <a href="https://github.com/codefornola/nola-transit-map">the README on GitHub</a> to learn more about how it works.
                        </p>
                    <Button onClick={handleClose}>
                    Close
                    </Button>
                </aside>
            </div>
            ) : (
                null
            )}
        </div>  
    )
}

export default CustomModal;