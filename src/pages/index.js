import React from 'react'
import Link from 'gatsby-link'
import Swiper from '../components/swiper'
import Menu from '../components/menu'
import {SectionsContainer, Section, Header, Footer} from 'react-fullpage';

let options = {
    sectionClassName:     'section',
    scrollBar:            false,
    navigation:           false,
    verticalAlign:        false,
    sectionPaddingTop:    '50px',
    sectionPaddingBottom: '50px',
};

const IndexPage = () => (
    <div id="outer-container">
        <SectionsContainer className="container" {...options}>
            <Header>
                <Menu />
            </Header>
            <Section id="page-wrap" color="#FFFFFF">
                <Swiper />
            </Section>
        </SectionsContainer>
    </div>
)

export default IndexPage
