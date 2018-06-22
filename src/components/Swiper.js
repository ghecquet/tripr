import React from 'react'
import ReactSwipe from 'react-swipe';
import styles from "./swiper.module.css";

export default () => (
    <ReactSwipe className={styles.mySwipe} swipeOptions={{continuous: false}}>
        <div><div className={styles.mySwipeItem}>PANE 1</div></div>
        <div><div className={styles.mySwipeItem}>PANE 2</div></div>
        <div><div className={styles.mySwipeItem}>PANE 3</div></div>
    </ReactSwipe>
);
