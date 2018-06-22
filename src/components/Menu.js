import React from 'react';
import {SectionsContainer, Section, Header, Footer} from 'react-fullpage';
import styles from "./menu.module.css";
import { CSSTransition } from 'react-transition-group';

// import { scaleDown as Menu } from 'react-burger-menu';

let options = {
    sectionClassName:     'section',
    anchors:              ['sectionOne', 'sectionTwo', 'sectionThree'],
    scrollBar:            false,
    navigation:           true,
    verticalAlign:        false,
    sectionPaddingTop:    '0px',
    sectionPaddingBottom: '0px',
};

class Menu extends React.Component {

    constructor(props) {
        super(props)

        this.binders = {
            "budget": new Binder(this.budgetMenuRightRef, this.budgetMainSectionRef)
        }

        this.state = {
            showMenu: true,
            showFullPageMenu: false,
            cssTransform: {
                "budget": this.binders["budget"].csstransform
            }
        }
    }

    handleClick(type) {

        this.setState({
            showFullPageMenu: !this.state.showFullPageMenu,
        })
    }

    handleAnimate(type) {
        let binder = new Binder(this.budgetMenuRightRef, this.budgetMainSectionRef)

            binder.join()

            this.setState({
                cssTransform: {
                    ...this.state.cssTransform,
                    "budget": binder.csstransform
                }
            })
    }

    render () {

        const {
            showMenu,
            showFullPageMenu
        } = this.state;

        return (
            <div>
                <CSSTransition
                    in={showFullPageMenu}
                    timeout={0}
                    className={showFullPageMenu ? styles.fullPageDisplay : styles.fullPageHide}
                    classNames={{
                        enter: styles.fullPageDisplay,
                        exit: styles.fullPageHide,
                    }}
                    onEntered={() => this.handleAnimate("budget")}
                >
                    <SectionsContainer className={styles.fullPageContainer} {...options}>
                        <Section className={styles.section} color="#69D2E7">
                            <h2>what is your <span ref={(el) => this.budgetMainSectionRef = el} style={{opacity: 0}}>budget</span> ?</h2>
                            <h3>400</h3>
                        </Section>
                        <Section className={styles.section} color="#A7DBD8">
                            <h2>where are you leaving from ?</h2>
                            <h3>New York</h3>
                        </Section>
                        <Section className={styles.section} color="#E0E4CC">
                        </Section>
                        <Section className={styles.section} color="#69D2E7">
                        </Section>
                    </SectionsContainer>
                </CSSTransition>

                <div id="menuRight" className={styles.menuRight}>
                    <div className={styles.menuRightItem} style={{"backgroundColor": "#69D2E7"}} onClick={() => this.handleClick("budget")}>
                        <h2><span ref={(el) => this.budgetMenuRightRef = el} className={styles.items} style={{transform: this.state.cssTransform.budget}}>budget</span></h2>
                    </div>
                    <div className={styles.menuRightItem} style={{"backgroundColor": "#A7DBD8"}}><h2><span>leaving from</span></h2></div>
                    <div className={styles.menuRightItem} style={{"backgroundColor": "#E0E4CC"}}></div>
                    <div className={styles.menuRightItem} style={{"backgroundColor": "#69D2E7"}}></div>
                </div>

            </div>
        );
    }
}

class Binder {
    constructor(source, target) {
        this.joined = true
        this.source = source
        this.target = target

        this.translateX = 0
        this.translateY = 0
    }

    get csstransform() {
        return "translate3D(" + this.translateX + "px," + this.translateY + "px,0)"
    }

    join() {
        if (!this.source || ! this.target) {
            return
        }

        const sourceRef = this.source.getBoundingClientRect()
        const targetRef = this.target.getBoundingClientRect()

        let sourceRefX = sourceRef.x - this.translateX
        let sourceRefY = sourceRef.y - this.translateY

        this.translateX = targetRef.x - sourceRefX
        this.translateY = targetRef.y - sourceRefY
    }

    reset() {
        this.translateX = 0
        this.translateY = 0
    }
}



export default Menu


// let options = {
//     sectionClassName:     'section',
//     anchors:              ['sectionOne', 'sectionTwo', 'sectionThree'],
//     scrollBar:            false,
//     navigation:           true,
//     verticalAlign:        false,
//     sectionPaddingTop:    '50px',
//     sectionPaddingBottom: '50px',
//     arrowNavigation:      true
// };
//
// export default () => (
//     <div style="width: 0px">
//         <SectionsContainer className="container" {...options}>
//             <Section color="#69D2E7">Page 1</Section>
//             <Section color="#A7DBD8">Page 2</Section>
//             <Section color="#E0E4CC">Page 3</Section>
//         </SectionsContainer>
//     </div>
// );
