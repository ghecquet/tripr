import React from 'react'
import Link from 'gatsby-link'
import Swiper from '../components/swiper'
import Menu from '../components/menu'
import {SectionsContainer, Section, Footer} from 'react-fullpage';
import Header from '../components/Header';
import Auth from '../containers/Auth';

let options = {
    sectionClassName:     'section',
    scrollBar:            false,
    navigation:           false,
    verticalAlign:        false,
    sectionPaddingTop:    '50px',
    sectionPaddingBottom: '50px',
};

const IndexPage = () => (
    <Auth>
    {auth => {
      return (
        <div id="outer-container">
            <Header
                background="background-image: linear-gradient(116deg, #08AEEA 0%, #2AF598 100%)"
                title={"test"}
                {...auth}
            />
            <Menu />
            <SectionsContainer className="container" {...options}>
                <Section id="page-wrap" color="#FFFFFF">
                    <Swiper />
                </Section>
            </SectionsContainer>
        </div>
      )
    }}
    </Auth>
);

export default IndexPage
