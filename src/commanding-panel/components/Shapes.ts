const Shapes: Record<string, { name: string; css: Record<string, string> }> = {
    rectangle: {
        name: 'Rectangle',
        css: {}
    },
    ellipse: {
      name: 'Ellipse',
      css: {
        borderRadius: '50%',
      },
    },
    bean: {
      name: 'Bean',
      css: {
        borderRadius: '9999px',
      },
    },
    arrowUp: {
      name: 'Up Arrow',
      css: {
        clipPath: 'polygon(50% 0%, 0% 100%, 100% 100%)',
      },
    },
    arrowDown: {
      name: 'Down Arrow',
      css: {
        clipPath: 'polygon(0% 0%, 100% 0%, 50% 100%)',
      },
    },
    arrowLeft: {
      name: 'Left Arrow',
      css: {
        clipPath: 'polygon(100% 0%, 0% 50%, 100% 100%)',
      },
    },
    arrowRight: {
      name: 'Right Arrow',
      css: {
        clipPath: 'polygon(0% 0%, 100% 50%, 0% 100%)',
      },
    },
    diamond: {
      name: 'Diamond',
      css: {
        clipPath: 'polygon(50% 0%, 100% 50%, 50% 100%, 0% 50%)',
      },
    },
  };

export default Shapes;
